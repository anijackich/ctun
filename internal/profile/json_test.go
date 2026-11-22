package profile

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"ctun/internal/config"
)

const xrayFull = `{
  "remarks": "Germany 🇩🇪",
  "outbounds": [
    {"protocol": "vless", "tag": "proxy",
     "settings": {"vnext": [{"address": "de.example.com", "port": 443,
       "users": [{"id": "00000000-0000-0000-0000-000000000000", "encryption": "none", "flow": ""}]}]},
     "streamSettings": {"network": "xhttp", "security": "reality",
       "realitySettings": {"serverName": "www.example.com", "publicKey": "AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA", "shortId": "ab", "fingerprint": "chrome"},
       "xhttpSettings": {"path": "/x", "mode": "stream-one"}}},
    {"protocol": "freedom", "tag": "direct"},
    {"protocol": "blackhole", "tag": "block"}
  ]
}`

const happArray = `[
  {"remarks": "NL", "outbounds": [{"protocol": "vless",
    "settings": {"vnext": [{"address": "nl.example.com", "port": 8443, "users": [{"id": "00000000-0000-0000-0000-000000000000", "flow": "xtls-rprx-vision"}]}]},
    "streamSettings": {"network": "tcp", "security": "reality", "realitySettings": {"serverName": "nl.example.com", "publicKey": "AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA", "shortId": ""}}}]},
  {"remarks": "FI", "outbounds": [{"protocol": "vmess",
    "settings": {"vnext": [{"address": "fi.example.com", "port": 443, "users": [{"id": "00000000-0000-0000-0000-000000000000", "alterId": 0}]}]},
    "streamSettings": {"network": "ws", "security": "tls", "tlsSettings": {"serverName": "fi.example.com"}, "wsSettings": {"path": "/ws", "headers": {"Host": "fi.example.com"}}}}]}
]`

const xrayOutbound = `{"protocol": "trojan", "tag": "Trojan WS",
  "settings": {"servers": [{"address": "tr.example.com", "port": 443, "password": "secret"}]},
  "streamSettings": {"network": "grpc", "security": "tls", "tlsSettings": {"serverName": "tr.example.com"}, "grpcSettings": {"serviceName": "svc"}}}`

const xrayFlat = `{"remarks": "Flat 🇳🇱", "outbounds": [
  {"protocol": "vless", "tag": "proxy",
   "settings": {"id": "00000000-0000-0000-0000-000000000000", "encryption": "none", "address": "192.0.2.1", "port": 443, "flow": "xtls-rprx-vision"},
   "streamSettings": {"network": "tcp", "security": "reality", "tcpSettings": {"header": {"type": "none"}},
     "realitySettings": {"serverName": "www.example.com", "publicKey": "AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA", "shortId": "ab", "fingerprint": "chrome", "spiderX": ""}}},
  {"protocol": "blackhole", "tag": "block"}, {"protocol": "freedom", "tag": "direct"}],
 "inbounds": [{"protocol": "socks", "port": 1080}]}`

const singBox = `{"outbounds": [
  {"type": "vless", "tag": "SB Reality", "server": "sb.example.com", "server_port": 443,
   "uuid": "00000000-0000-0000-0000-000000000000", "flow": "xtls-rprx-vision",
   "tls": {"enabled": true, "server_name": "sb.example.com", "utls": {"enabled": true, "fingerprint": "chrome"},
           "reality": {"enabled": true, "public_key": "AQIDBAUGBwgJCgsMDQ4PEBESExQVFhcYGRobHB0eHyA", "short_id": "cd"}}},
  {"type": "shadowsocks", "tag": "SS", "server": "ss.example.com", "server_port": 8388, "method": "2022-blake3-aes-128-gcm", "password": "AAAAAAAAAAAAAAAAAAAAAA=="},
  {"type": "direct", "tag": "direct"}
]}`

func TestParseJSONFormats(t *testing.T) {
	cases := map[string]struct {
		body  string
		names []string
	}{
		"xray full config": {xrayFull, []string{"Germany 🇩🇪"}},
		"happ array":       {happArray, []string{"NL", "FI"}},
		"bare outbound":    {xrayOutbound, []string{"Trojan WS"}},
		"xray flat":        {xrayFlat, []string{"Flat 🇳🇱"}},
		"sing-box":         {singBox, []string{"SB Reality", "SS"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			layer, err := Parse([]byte(tc.body))
			if err != nil {
				t.Fatal(err)
			}
			if len(layer.Proxies) != len(tc.names) {
				t.Fatalf("got %d proxies, want %d", len(layer.Proxies), len(tc.names))
			}
			for i, want := range tc.names {
				if got := layer.Proxies[i]["name"]; got != want {
					t.Errorf("proxy %d name = %v, want %s", i, got, want)
				}
			}
			checkWithCore(t, layer)
		})
	}
}

func TestXrayRealityXHTTPMapping(t *testing.T) {
	layer, err := Parse([]byte(xrayFull))
	if err != nil {
		t.Fatal(err)
	}
	p := layer.Proxies[0]
	if got := Describe(p); got != "VLESS · Reality · xhttp" {
		t.Errorf("describe = %q", got)
	}
	if p["servername"] != "www.example.com" || p["client-fingerprint"] != "chrome" {
		t.Errorf("tls fields = %v", p)
	}
	if opts := p["xhttp-opts"].(map[string]any); opts["path"] != "/x" || opts["mode"] != "stream-one" {
		t.Errorf("xhttp-opts = %v", opts)
	}
}

func TestTrojanUsesSNI(t *testing.T) {
	layer, err := Parse([]byte(xrayOutbound))
	if err != nil {
		t.Fatal(err)
	}
	if p := layer.Proxies[0]; p["sni"] != "tr.example.com" || p["servername"] != nil {
		t.Errorf("trojan must use sni: %v", p)
	}
}

func checkWithCore(t *testing.T, layer *config.Profile) {
	t.Helper()
	core := filepath.Join("..", "..", "core", "mihomo-"+runtime.GOOS+"-"+runtime.GOARCH)
	if _, err := os.Stat(core); err != nil {
		return
	}
	cfg, err := config.Build(config.Options{
		RoutingMode: "rule",
		LogLevel:    "warning",
		Controller:  config.Controller{Addr: "127.0.0.1:1", Secret: "x"},
		Profile:     layer,
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, cfg, 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.CommandContext(t.Context(), core, "-t", "-d", dir, "-f", path).CombinedOutput(); err != nil {
		t.Fatalf("core rejected generated proxies: %s", out)
	}
}
