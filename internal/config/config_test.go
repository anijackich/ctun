package config

import (
	"errors"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ctun/internal/routing"
)

var update = flag.Bool("update", false, "rewrite golden files")

func golden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden.yaml")
	if *update {
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("%s mismatch:\n--- got\n%s\n--- want\n%s", name, got, want)
	}
}

func TestBuildProxyMode(t *testing.T) {
	out, err := Build(Options{
		RoutingMode: "rule",
		LogLevel:    "warning",
		MixedPort:   7890,
		Auth:        &Credentials{User: "ctun", Password: "pass"},
		Controller:  Controller{Addr: "127.0.0.1:41234", Secret: "s3cret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "proxy", out)
}

func TestBuildWithoutPortOmitsListener(t *testing.T) {
	out, err := Build(Options{
		RoutingMode: "global",
		LogLevel:    "info",
		Controller:  Controller{Addr: "127.0.0.1:41234", Secret: "s3cret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "no-port", out)
}

func TestBuildRejectsPortWithoutAuth(t *testing.T) {
	_, err := Build(Options{MixedPort: 7890, Controller: Controller{Addr: "127.0.0.1:1"}})
	if !errors.Is(err, ErrPortWithoutAuth) {
		t.Fatalf("err = %v, want ErrPortWithoutAuth", err)
	}
}

func ruRouting(p *Profile) *Routing {
	profile := routing.Profile{Name: "RU", Final: routing.Proxy, Lists: []routing.List{
		{Target: routing.Direct, Entries: []string{"geosite:category-ru", "geoip:ru"}},
	}}
	c := routing.Compile(profile, MainGroup(p))
	return &Routing{Rules: c.Rules, Providers: c.Providers, Final: c.Final}
}

func TestBuildLinkListProfileWithUserRouting(t *testing.T) {
	p := &Profile{Proxies: []map[string]any{
		{"name": "🇩🇪 DE", "type": "vless", "server": "de.example.com", "port": 443, "uuid": "00000000-0000-0000-0000-000000000000"},
		{"name": "🇫🇮 FI", "type": "vless", "server": "fi.example.com", "port": 443, "uuid": "00000000-0000-0000-0000-000000000000"},
	}}
	out, err := Build(Options{
		RoutingMode: "rule",
		LogLevel:    "warning",
		Controller:  Controller{Addr: "127.0.0.1:41234", Secret: "s3cret"},
		Profile:     p,
		Routing:     ruRouting(p),
	})
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "links-profile", out)
}

func TestBuildKeepsProfileOwnRouting(t *testing.T) {
	out, err := Build(Options{
		RoutingMode: "rule",
		LogLevel:    "warning",
		Controller:  Controller{Addr: "127.0.0.1:41234", Secret: "s3cret"},
		Profile: &Profile{
			Proxies:     []map[string]any{{"name": "a", "type": "socks5", "server": "127.0.0.1", "port": 1080}},
			ProxyGroups: []map[string]any{{"name": "Main", "type": "select", "proxies": []any{"a", "DIRECT"}}},
			Rules:       []string{"DOMAIN-SUFFIX,example.com,DIRECT", "MATCH,Main"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "clash-profile", out)
}

func TestGroupsDefaultAndOwn(t *testing.T) {
	links := Groups(&Profile{Proxies: []map[string]any{{"name": "x"}, {"name": "y"}}})
	if len(links) != 2 || links[0].Name != PrimaryGroup || links[0].Proxies[0] != AutoGroup || links[1].Proxies[0] != "x" {
		t.Fatalf("default groups = %+v", links)
	}
	single := Groups(&Profile{Proxies: []map[string]any{{"name": "x"}}})
	if len(single) != 1 || len(single[0].Proxies) != 1 || single[0].Proxies[0] != "x" {
		t.Fatalf("single-server groups = %+v", single)
	}
	own := Groups(&Profile{ProxyGroups: []map[string]any{{"name": "Main", "type": "select", "proxies": []any{"a"}}}})
	if len(own) != 1 || own[0].Name != "Main" || own[0].Proxies[0] != "a" {
		t.Fatalf("own groups = %+v", own)
	}
}

func TestBuildTUNHasNoListenerAndFakeIPDNS(t *testing.T) {
	p := &Profile{Proxies: []map[string]any{
		{"name": "🇩🇪 DE", "type": "socks5", "server": "de.example.com", "port": 1080},
	}}
	out, err := Build(Options{
		RoutingMode: "rule",
		LogLevel:    "warning",
		Controller:  Controller{Addr: "127.0.0.1:41234", Secret: "s3cret"},
		TUN:         &TUN{Stack: "mixed"},
		Profile:     p,
		Routing:     ruRouting(p),
	})
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "tun", out)
	if strings.Contains(string(out), "mixed-port") {
		t.Error("TUN mode must not open a local proxy port")
	}
}

func TestRulesPutUserProfileBeforeSubscription(t *testing.T) {
	sub := &Profile{
		ProxyGroups:   []map[string]any{{"name": "Main", "type": "select", "proxies": []any{"a"}}},
		Rules:         []string{"RULE-SET,ads,REJECT", "DOMAIN-SUFFIX,example.com,DIRECT", "MATCH,Main"},
		RuleProviders: map[string]any{"geosite-youtube": map[string]any{"type": "http"}},
	}
	user := &Routing{
		Rules:     []string{"RULE-SET,geosite-youtube,Main", "PROCESS-NAME,Telegram,DIRECT"},
		Providers: map[string]any{"geosite-youtube": map[string]any{"type": "http", "url": "ours"}},
		Final:     "DIRECT",
	}
	rules, providers := Rules(sub, user)
	want := []string{
		lanRule,
		"RULE-SET,ctun-geosite-youtube,Main",
		"PROCESS-NAME,Telegram,DIRECT",
		"RULE-SET,ads,REJECT",
		"DOMAIN-SUFFIX,example.com,DIRECT",
		"MATCH,DIRECT",
	}
	if strings.Join(rules, "\n") != strings.Join(want, "\n") {
		t.Fatalf("rules =\n%s", strings.Join(rules, "\n"))
	}
	if providers["ctun-geosite-youtube"].(map[string]any)["url"] != "ours" || providers["geosite-youtube"].(map[string]any)["url"] != nil {
		t.Fatalf("providers = %v", providers)
	}

	rules, _ = Rules(sub, nil)
	if rules[len(rules)-1] != "MATCH,Main" {
		t.Fatalf("without a user profile the subscription's catch-all stays: %v", rules)
	}
}

func TestMainGroup(t *testing.T) {
	if got := MainGroup(&Profile{Proxies: []map[string]any{{"name": "a"}}}); got != PrimaryGroup {
		t.Errorf("link list → %q", got)
	}
	own := &Profile{ProxyGroups: []map[string]any{
		{"name": "Auto", "type": "url-test"}, {"name": "🚀 Select", "type": "select"},
	}}
	if got := MainGroup(own); got != "🚀 Select" {
		t.Errorf("own groups → %q", got)
	}
}

func TestProxyLANDropsLocalNetworkRule(t *testing.T) {
	p := &Profile{Proxies: []map[string]any{{"name": "a"}}}
	rules, _ := Rules(p, &Routing{Final: "PROXY", ProxyLAN: true})
	if len(rules) != 1 || rules[0] != "MATCH,PROXY" {
		t.Fatalf("rules = %v", rules)
	}
	if rules, _ = Rules(p, &Routing{Final: "PROXY"}); rules[0] != lanRule {
		t.Fatalf("local network must go direct by default: %v", rules)
	}
}
