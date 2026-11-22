package routing

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompiledRulesPassCore(t *testing.T) {
	core := filepath.Join("..", "..", "core", "mihomo-"+runtime.GOOS+"-"+runtime.GOARCH)
	if _, err := os.Stat(core); err != nil {
		t.Skip("core not fetched")
	}
	p := Profile{Name: "All", Final: Block, Lists: []List{
		{Target: Block, Entries: []string{"geosite:category-ads-all", "regexp:^ads?\\.", "port:6881-6889"}},
		{Target: Direct, Entries: []string{"geosite:category-ru", "geoip:ru", "geoip:private", "full:www.example.com", "1.2.3.4", "2001:db8::/32", "process:Telegram"}},
		{Target: Proxy, Entries: []string{"youtube.com", "keyword:google", "DOMAIN-SUFFIX,example.org", "AND,((DOMAIN,a.com),(NETWORK,UDP))"}},
	}}
	c := Compile(p, "PROXY")
	if len(c.Errors) != 0 {
		t.Fatalf("errors: %+v", c.Errors)
	}
	var b strings.Builder
	b.WriteString("mode: rule\nproxies: []\nproxy-groups:\n  - {name: PROXY, type: select, proxies: [DIRECT]}\nrule-providers:\n")
	for name, raw := range c.Providers {
		set := raw.(map[string]any)
		b.WriteString("  " + name + ": {type: http, behavior: " + set["behavior"].(string) + ", format: mrs, url: '" + set["url"].(string) + "', path: '" + set["path"].(string) + "', interval: 86400}\n")
	}
	b.WriteString("rules:\n")
	for _, r := range append(c.Rules, "MATCH,"+c.Final) {
		b.WriteString("  - '" + r + "'\n")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(b.String()), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.CommandContext(t.Context(), core, "-t", "-d", dir, "-f", path).CombinedOutput(); err != nil {
		t.Fatalf("core rejected compiled rules: %s\n%s", out, b.String())
	}
}
