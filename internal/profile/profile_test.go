package profile

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const links = "vless://00000000-0000-0000-0000-000000000000@de.example.com:8443?type=xhttp&security=reality&pbk=AAAA&sid=ab&sni=de.example.com&mode=stream-one#Germany%20%F0%9F%87%A9%F0%9F%87%AA\n" +
	"vless://00000000-0000-0000-0000-000000000000@fi.example.com:443?type=ws&security=tls&sni=fi.example.com&path=%2Fws#Finland%20%F0%9F%87%AB%F0%9F%87%AE\n" +
	"vless://00000000-0000-0000-0000-000000000000@fi2.example.com:443?type=ws&security=tls&sni=fi.example.com&path=%2Fws#Finland%20%F0%9F%87%AB%F0%9F%87%AE\n"

const clashYAML = `
proxies:
  - {name: a, type: socks5, server: 127.0.0.1, port: 1080}
proxy-groups:
  - {name: Main, type: select, proxies: [a, DIRECT]}
rules:
  - MATCH,Main
`

func gz(t *testing.T, data []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := gzip.NewWriter(&buf)
	if _, err := w.Write(data); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestParseFormats(t *testing.T) {
	b64 := []byte(base64.StdEncoding.EncodeToString([]byte(links)))
	for name, body := range map[string][]byte{
		"plain links":      []byte(links),
		"base64 links":     b64,
		"gzipped base64":   gz(t, b64),
		"wrapped base64":   []byte(string(b64[:40]) + "\n" + string(b64[40:])),
		"links with a BOM": append([]byte("\xef\xbb\xbf"), links...),
	} {
		t.Run(name, func(t *testing.T) {
			p, err := Parse(body)
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Proxies) != 3 {
				t.Fatalf("got %d proxies", len(p.Proxies))
			}

			if p.Proxies[0]["name"] != "Germany 🇩🇪" || p.Proxies[2]["name"] != "Finland 🇫🇮-01" {
				t.Fatalf("names = %v, %v", p.Proxies[0]["name"], p.Proxies[2]["name"])
			}
			if len(p.ProxyGroups) != 0 {
				t.Fatalf("link lists must not carry groups")
			}
		})
	}
}

func TestParseClashKeepsRouting(t *testing.T) {
	p, err := Parse([]byte(clashYAML))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Proxies) != 1 || len(p.ProxyGroups) != 1 || p.Rules[0] != "MATCH,Main" {
		t.Fatalf("parsed = %+v", p)
	}
}

func TestParseEmptyIsAnError(t *testing.T) {
	for _, body := range []string{"", "proxies: []", "not a subscription"} {
		if _, err := Parse([]byte(body)); !errors.Is(err, ErrNoProxies) {
			t.Errorf("Parse(%q) err = %v, want ErrNoProxies", body, err)
		}
	}
}

func TestDescribe(t *testing.T) {
	p, err := Parse([]byte(links))
	if err != nil {
		t.Fatal(err)
	}
	if got := Describe(p.Proxies[0]); got != "VLESS · Reality · xhttp" {
		t.Errorf("reality = %q", got)
	}
	if got := Describe(p.Proxies[1]); got != "VLESS · TLS · ws" {
		t.Errorf("tls = %q", got)
	}
}

func TestParseHeaders(t *testing.T) {
	h := http.Header{}
	h.Set("Subscription-Userinfo", "upload=10; download=20; total=1073741824; expire=1802965752")
	h.Set("Profile-Title", "base64:"+base64.StdEncoding.EncodeToString([]byte("Example VPN")))
	h.Set("Profile-Update-Interval", "12")
	h.Set("Support-Url", " https://t.me/support ")
	h.Set("Announce", "base64:"+base64.StdEncoding.EncodeToString([]byte("Привет")))

	m := ParseHeaders(h)
	want := Info{Upload: 10, Download: 20, Total: 1 << 30, Expire: 1802965752, SupportURL: "https://t.me/support", Announce: "Привет"}
	if m.Info != want || m.Title != "Example VPN" || m.UpdateIntervalHours != 12 {
		t.Fatalf("meta = %+v", m)
	}

	h = http.Header{}
	h.Set("Content-Disposition", `attachment; filename="My Provider.yaml"`)
	if got := ParseHeaders(h).Title; got != "My Provider" {
		t.Errorf("title from filename = %q", got)
	}
}

func TestFetchFallsBackToCoreProxyWithoutLeakingURL(t *testing.T) {
	const secretPath = "/sub/SECRET-TOKEN"
	origin := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "test-ua" {
			http.Error(w, "ua", http.StatusBadRequest)
			return
		}
		_, _ = w.Write([]byte(links))
	}))
	defer origin.Close()

	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, _ := http.NewRequestWithContext(r.Context(), http.MethodGet, origin.URL+r.URL.Path, nil)
		req.Header.Set("User-Agent", r.Header.Get("User-Agent"))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		defer resp.Body.Close()
		var buf bytes.Buffer
		_, _ = buf.ReadFrom(resp.Body)
		_, _ = w.Write(buf.Bytes())
	}))
	defer proxy.Close()

	target := "http://unreachable.invalid" + secretPath
	f := &Fetcher{}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := f.Fetch(ctx, target, "test-ua"); err == nil {
		t.Fatal("direct fetch of an invalid host must fail")
	} else if strings.Contains(err.Error(), "SECRET-TOKEN") {
		t.Fatalf("error leaks the subscription URL: %v", err)
	}

	f.ProxyURL = func() string { return proxy.URL }
	resp, err := f.Fetch(ctx, target, "test-ua")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(resp.Body), "vless://") {
		t.Fatalf("body = %q", resp.Body)
	}
}

func TestStoreLifecycle(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "profiles")
	s, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	a := Profile{ID: "a", Name: "A", URL: "https://a.example.com/sub/x"}
	b := Profile{ID: "b", Name: "B", URL: "https://b.example.com/sub/y"}
	if err := s.Put(a, []byte(links)); err != nil {
		t.Fatal(err)
	}
	if err := s.Put(b, []byte(clashYAML)); err != nil {
		t.Fatal(err)
	}
	if s.ActiveID() != "a" {
		t.Fatalf("first profile must become active, got %q", s.ActiveID())
	}

	if _, err := s.Update("b", func(p *Profile) { p.Selected = map[string]string{"Main": "a"} }); err != nil {
		t.Fatal(err)
	}
	reopened, err := OpenStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	got, err := reopened.Get("b")
	if err != nil || got.Selected["Main"] != "a" {
		t.Fatalf("reloaded b = %+v, %v", got, err)
	}
	if layer, err := reopened.Layer("a"); err != nil || len(layer.Proxies) != 3 {
		t.Fatalf("layer a = %v, %v", layer, err)
	}

	if err := reopened.Remove("a"); err != nil {
		t.Fatal(err)
	}
	if reopened.ActiveID() != "b" || len(reopened.List()) != 1 {
		t.Fatalf("after remove: active %q, %d profiles", reopened.ActiveID(), len(reopened.List()))
	}
	if _, err := reopened.Layer("a"); err == nil {
		t.Fatal("subscription file must be removed")
	}
}

func TestDue(t *testing.T) {
	now := time.Now()
	p := Profile{UpdateIntervalHours: 2, UpdatedAt: now.Add(-3 * time.Hour)}
	if !p.Due(now) {
		t.Error("3h old with 2h interval must be due")
	}
	p.UpdateIntervalHours = 0
	if p.Due(now) {
		t.Error("manual profiles are never due")
	}
}

func TestDedupeNamesForClashProfiles(t *testing.T) {
	proxies := []map[string]any{{"name": "x"}, {"name": "x"}, {"name": "x 2"}, {"server": "h", "port": 1}}
	dedupeNames(proxies)
	got := []any{proxies[0]["name"], proxies[1]["name"], proxies[2]["name"], proxies[3]["name"]}
	want := []any{"x", "x 2", "x 2 2", "h:1"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("names = %v, want %v", got, want)
		}
	}
}

func TestIsLink(t *testing.T) {
	for text, want := range map[string]bool{
		"https://sub.example.com/abc":              true,
		"http://sub.example.com":                   true,
		"vless://id@host:443":                      false,
		"https://a.example.com\nhttps://b.example": false,
		"https://":          false,
		`{"outbounds": []}`: false,
	} {
		if got := IsLink(text); got != want {
			t.Errorf("IsLink(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestReadConfigRejectsHugeFiles(t *testing.T) {
	dir := t.TempDir()
	small, huge := filepath.Join(dir, "small.json"), filepath.Join(dir, "huge.json")
	if err := os.WriteFile(small, []byte(`{"outbounds": []}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(huge, make([]byte, maxImportSize+1), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadConfig(small); err != nil {
		t.Errorf("small file: %v", err)
	}
	if _, err := ReadConfig(huge); err == nil {
		t.Error("files over the limit must be rejected")
	}
}
