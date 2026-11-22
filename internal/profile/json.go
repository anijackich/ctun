package profile

import (
	"cmp"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func parseJSON(body []byte) []map[string]any {
	var root any
	if err := json.Unmarshal(body, &root); err != nil {
		return nil
	}
	var proxies []map[string]any
	for _, doc := range asList(root) {
		obj, ok := doc.(map[string]any)
		if !ok {
			continue
		}
		remarks := str(obj["remarks"])
		outbounds, hasList := obj["outbounds"].([]any)
		if !hasList {
			outbounds = []any{obj}
		}
		for _, raw := range outbounds {
			outbound, ok := raw.(map[string]any)
			if !ok {
				continue
			}
			var proxy map[string]any
			if _, isXray := outbound["protocol"]; isXray {
				proxy = fromXray(outbound)
			} else {
				proxy = fromSingBox(outbound)
			}
			if proxy == nil {
				continue
			}
			proxy["name"] = proxyName(remarks, outbound, proxy)
			proxies = append(proxies, proxy)
		}
	}
	return proxies
}

func asList(v any) []any {
	if list, ok := v.([]any); ok {
		return list
	}
	return []any{v}
}

func proxyName(remarks string, outbound, proxy map[string]any) string {
	tag := str(outbound["tag"])
	switch {
	case remarks != "":
		return remarks
	case tag != "" && tag != "proxy":
		return tag
	default:
		return fmt.Sprintf("%v:%v", proxy["server"], proxy["port"])
	}
}

func fromXray(o map[string]any) map[string]any {
	settings := obj(o["settings"])
	proxy := map[string]any{"udp": true}
	switch protocol := str(o["protocol"]); protocol {
	case "vless", "vmess":
		server := firstOr(settings["vnext"], settings)
		user := firstOr(server["users"], server)
		proxy["type"] = protocol
		proxy["server"], proxy["port"] = server["address"], num(server["port"])
		proxy["uuid"] = user["id"]
		if protocol == "vless" {
			setIf(proxy, "flow", user["flow"])
		} else {
			proxy["alterId"] = num(user["alterId"])
			proxy["cipher"] = cmp.Or(str(user["security"]), "auto")
		}
	case "trojan", "shadowsocks":
		server := firstOr(settings["servers"], settings)
		proxy["server"], proxy["port"] = server["address"], num(server["port"])
		proxy["password"] = server["password"]
		if protocol == "trojan" {
			proxy["type"] = "trojan"
		} else {
			proxy["type"] = "ss"
			proxy["cipher"] = server["method"]
		}
	default:
		return nil
	}
	if proxy["server"] == nil {
		return nil
	}
	applyXrayStream(proxy, obj(o["streamSettings"]))
	return proxy
}

func applyXrayStream(proxy, stream map[string]any) {
	network := cmp.Or(str(stream["network"]), "tcp")
	switch str(stream["security"]) {
	case "reality":
		r := obj(stream["realitySettings"])
		proxy["tls"] = true
		setServerName(proxy, r["serverName"])
		setIf(proxy, "client-fingerprint", r["fingerprint"])
		proxy["reality-opts"] = map[string]any{"public-key": r["publicKey"], "short-id": str(r["shortId"])}
	case "tls":
		t := obj(stream["tlsSettings"])
		proxy["tls"] = true
		setServerName(proxy, t["serverName"])
		setIf(proxy, "client-fingerprint", t["fingerprint"])
		setIf(proxy, "alpn", t["alpn"])
		if t["allowInsecure"] == true {
			proxy["skip-cert-verify"] = true
		}
	}

	switch network {
	case "ws":
		ws := obj(stream["wsSettings"])
		opts := map[string]any{"path": cmp.Or(str(ws["path"]), "/")}
		if host := cmp.Or(str(ws["host"]), str(obj(ws["headers"])["Host"])); host != "" {
			opts["headers"] = map[string]any{"Host": host}
		}
		proxy["network"], proxy["ws-opts"] = "ws", opts
	case "httpupgrade":
		h := obj(stream["httpupgradeSettings"])
		opts := map[string]any{"path": cmp.Or(str(h["path"]), "/"), "v2ray-http-upgrade": true}
		if host := str(h["host"]); host != "" {
			opts["headers"] = map[string]any{"Host": host}
		}
		proxy["network"], proxy["ws-opts"] = "ws", opts
	case "grpc":
		proxy["network"] = "grpc"
		proxy["grpc-opts"] = map[string]any{"grpc-service-name": str(obj(stream["grpcSettings"])["serviceName"])}
	case "xhttp", "splithttp":
		x := obj(stream["xhttpSettings"])
		if len(x) == 0 {
			x = obj(stream["splithttpSettings"])
		}
		opts := map[string]any{"path": cmp.Or(str(x["path"]), "/")}
		setIf(opts, "host", x["host"])
		setIf(opts, "mode", x["mode"])
		proxy["network"], proxy["xhttp-opts"] = "xhttp", opts
	default:
		proxy["network"] = "tcp"
	}
}

func fromSingBox(o map[string]any) map[string]any {
	proxy := map[string]any{"udp": true, "server": o["server"], "port": num(o["server_port"])}
	switch kind := str(o["type"]); kind {
	case "vless":
		proxy["type"], proxy["uuid"] = "vless", o["uuid"]
		setIf(proxy, "flow", o["flow"])
	case "vmess":
		proxy["type"], proxy["uuid"] = "vmess", o["uuid"]
		proxy["alterId"] = num(o["alter_id"])
		proxy["cipher"] = cmp.Or(str(o["security"]), "auto")
	case "trojan":
		proxy["type"], proxy["password"] = "trojan", o["password"]
	case "shadowsocks":
		proxy["type"], proxy["cipher"], proxy["password"] = "ss", o["method"], o["password"]
	default:
		return nil
	}
	if proxy["server"] == nil {
		return nil
	}

	if tls := obj(o["tls"]); tls["enabled"] == true {
		proxy["tls"] = true
		setServerName(proxy, tls["server_name"])
		setIf(proxy, "alpn", tls["alpn"])
		setIf(proxy, "client-fingerprint", obj(tls["utls"])["fingerprint"])
		if tls["insecure"] == true {
			proxy["skip-cert-verify"] = true
		}
		if reality := obj(tls["reality"]); reality["enabled"] == true {
			proxy["reality-opts"] = map[string]any{"public-key": reality["public_key"], "short-id": str(reality["short_id"])}
		}
	}

	transport := obj(o["transport"])
	switch str(transport["type"]) {
	case "ws":
		opts := map[string]any{"path": cmp.Or(str(transport["path"]), "/")}
		if host := str(obj(transport["headers"])["Host"]); host != "" {
			opts["headers"] = map[string]any{"Host": host}
		}
		proxy["network"], proxy["ws-opts"] = "ws", opts
	case "grpc":
		proxy["network"] = "grpc"
		proxy["grpc-opts"] = map[string]any{"grpc-service-name": str(transport["service_name"])}
	case "httpupgrade":
		opts := map[string]any{"path": cmp.Or(str(transport["path"]), "/"), "v2ray-http-upgrade": true}
		setIf(opts, "host", transport["host"])
		proxy["network"], proxy["ws-opts"] = "ws", opts
	default:
		proxy["network"] = "tcp"
	}
	return proxy
}

func setServerName(proxy map[string]any, name any) {
	if str(name) == "" {
		return
	}
	if proxy["type"] == "trojan" {
		proxy["sni"] = name
	} else {
		proxy["servername"] = name
	}
}

func obj(v any) map[string]any {
	m, _ := v.(map[string]any)
	if m == nil {
		return map[string]any{}
	}
	return m
}

func firstOr(v any, flat map[string]any) map[string]any {
	if list, ok := v.([]any); ok && len(list) > 0 {
		return obj(list[0])
	}
	return flat
}

func num(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case string:
		i, _ := strconv.Atoi(strings.TrimSpace(n))
		return i
	}
	return 0
}

func setIf(m map[string]any, key string, v any) {
	switch val := v.(type) {
	case nil:
	case string:
		if val != "" {
			m[key] = val
		}
	case []any:
		if len(val) > 0 {
			m[key] = val
		}
	default:
		m[key] = val
	}
}
