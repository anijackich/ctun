package routing

import (
	"slices"
	"testing"
)

func TestParseEntry(t *testing.T) {
	ok := map[string]string{
		"geosite:category-ads-all":           "RULE-SET,geosite-category-ads-all",
		"GeoSite:Geolocation-!cn":            "RULE-SET,geosite-geolocation-!cn",
		"geoip:RU":                           "RULE-SET,geoip-ru",
		"geoip:private":                      "GEOIP,LAN,no-resolve",
		"domain:Example.com":                 "DOMAIN-SUFFIX,example.com",
		"example.com":                        "DOMAIN-SUFFIX,example.com",
		"*.example.com":                      "DOMAIN-SUFFIX,example.com",
		"full:www.example.com":               "DOMAIN,www.example.com",
		"domain:ru":                          "DOMAIN-SUFFIX,ru",
		"domain:.xn--p1ai":                   "DOMAIN-SUFFIX,xn--p1ai",
		"keyword:google":                     "DOMAIN-KEYWORD,google",
		`regexp:^ads?\.`:                     `DOMAIN-REGEX,^ads?\.`,
		"1.2.3.4":                            "IP-CIDR,1.2.3.4/32,no-resolve",
		"ip:10.1.2.3/8":                      "IP-CIDR,10.0.0.0/8,no-resolve",
		"2001:db8::/32":                      "IP-CIDR6,2001:db8::/32,no-resolve",
		"process:Telegram":                   "PROCESS-NAME,Telegram",
		"process:/Applications/Telegram.app": "PROCESS-PATH,/Applications/Telegram.app",
		"port:8000-9000":                     "DST-PORT,8000-9000",
		"DOMAIN-SUFFIX,example.org":          "DOMAIN-SUFFIX,example.org",
		"IP-CIDR,192.0.2.0/24,no-resolve":    "IP-CIDR,192.0.2.0/24,no-resolve",
		"GEOSITE,youtube":                    "RULE-SET,geosite-youtube",
		"AND,((DOMAIN,a.com),(NETWORK,UDP))": "AND,((DOMAIN,a.com),(NETWORK,UDP))",
		"  # comment":                        "",
		"":                                   "",
	}
	for raw, want := range ok {
		r, err := parseEntry(raw)
		if err != nil {
			t.Errorf("%q: %v", raw, err)
			continue
		}
		if got := r.match + r.params; got != want {
			t.Errorf("%q → %q, want %q", raw, got, want)
		}
	}
	for _, raw := range []string{
		"geosite:", "geosite:bad name", "domain:not_a_domain", "regexp:(", "port:http",
		"wat:thing", "DOMAIN-SUFFIX,x.com,DIRECT", "RULE-SET,custom", "AND,((DOMAIN,a)),DIRECT",
		"keyword:a,b", "just words", "ru", `regexp:.*\\.ru$`,
	} {
		if _, err := parseEntry(raw); err == nil {
			t.Errorf("%q must be rejected", raw)
		}
	}
}

func TestCompileDefaultHasNoRulesOfOurs(t *testing.T) {
	c := Compile(Default(), "PROXY")
	if len(c.Rules) != 0 || len(c.Providers) != 0 || c.Final != "PROXY" || c.ProxyLAN {
		t.Fatalf("compiled = %+v", c)
	}
}

func TestCompileCategoryRuleSet(t *testing.T) {
	c := Compile(Profile{Name: "RU", Lists: []List{{Target: Direct, Entries: []string{"geosite:category-ru"}}}, ProxyLAN: true}, "PROXY")
	set := c.Providers["geosite-category-ru"].(map[string]any)
	if set["behavior"] != "domain" || set["proxy"] != "PROXY" || set["path"] != "./rules/geosite/category-ru.mrs" || !c.ProxyLAN {
		t.Fatalf("compiled = %+v", c)
	}
}

func TestCompileOrderTargetsAndErrors(t *testing.T) {
	p := Profile{
		Name: "Custom",
		Lists: []List{
			{Target: Proxy, Entries: []string{"youtube.com"}},
			{Target: Block, Entries: []string{"geosite:category-ads-all", "nonsense here"}},
			{Target: Direct, Entries: []string{"process:Telegram"}},
		},
		Final: Direct,
	}
	c := Compile(p, "🚀 Main")
	want := []string{
		"DOMAIN-SUFFIX,youtube.com,🚀 Main",
		"RULE-SET,geosite-category-ads-all,REJECT",
		"PROCESS-NAME,Telegram,DIRECT",
	}
	if !slices.Equal(c.Rules, want) || c.Final != "DIRECT" {
		t.Fatalf("rules = %q, final = %q", c.Rules, c.Final)
	}
	if len(c.Errors) != 1 || c.Errors[0].Target != Block || c.Errors[0].Line != 2 {
		t.Fatalf("errors = %+v", c.Errors)
	}
	if c.Providers["geosite-category-ads-all"].(map[string]any)["proxy"] != "🚀 Main" {
		t.Fatal("rule sets must download through the proxy group")
	}
}

func TestNormalizeKeepsOneListPerTarget(t *testing.T) {
	p := Profile{Lists: []List{
		{Target: Direct, Entries: []string{"a.com", "", "  "}},
		{Target: Direct, Entries: []string{"dup.com"}},
		{Target: "bogus"},
	}, Final: "bogus"}.normalize()
	if len(p.Lists) != 3 || p.Lists[0].Target != Direct || p.Lists[1].Target != Block || p.Lists[2].Target != Proxy {
		t.Fatalf("lists = %+v", p.Lists)
	}
	if !slices.Equal(p.Lists[0].Entries, []string{"a.com"}) || p.Final != Proxy {
		t.Fatalf("profile = %+v", p)
	}
}
