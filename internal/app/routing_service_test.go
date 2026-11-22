package app

import (
	"strings"
	"testing"

	"ctun/internal/routing"
)

func TestParseRule(t *testing.T) {
	cases := map[string]RuleView{
		"RULE-SET,ru-domains,DIRECT":                {Type: "RULE-SET", Payload: "ru-domains", Target: "DIRECT"},
		"GEOIP,LAN,DIRECT,no-resolve":               {Type: "GEOIP", Payload: "LAN", Target: "DIRECT"},
		"MATCH,PROXY":                               {Type: "MATCH", Target: "PROXY"},
		"AND,((DOMAIN,a.com),(NETWORK,UDP)),REJECT": {Type: "AND", Payload: "((DOMAIN,a.com),(NETWORK,UDP))", Target: "REJECT"},
		"domain-suffix, example.com , Proxy":        {Type: "DOMAIN-SUFFIX", Payload: "example.com", Target: "Proxy"},
	}
	for raw, want := range cases {
		if got := parseRule(0, raw); got != want {
			t.Errorf("%q → %+v, want %+v", raw, got, want)
		}
	}
}

func TestConfiguredRoutingFollowsActiveProfile(t *testing.T) {
	s := newTestService(t, proxyMode, true)
	rs := NewRoutingService(s, s.profiles, s.routes)
	view, err := rs.Routing()
	if err != nil {
		t.Fatal(err)
	}
	types := make([]string, len(view.Rules))
	for i, r := range view.Rules {
		types[i] = r.Type + ":" + r.Payload + ":" + r.Target
	}
	want := "GEOIP:LAN:DIRECT MATCH::PROXY"
	if got := strings.Join(types, " "); view.Live || got != want || len(view.RuleSets) != 0 {
		t.Fatalf("rules = %s, rule sets = %+v", got, view.RuleSets)
	}

	work, err := rs.CreateProfile("Work", "")
	if err != nil {
		t.Fatal(err)
	}
	work.Profile.Lists[0].Entries = []string{"geosite:category-ads-all", "not valid at all"}
	work.Profile.Final = routing.Direct
	saved, err := rs.SaveProfile(work.Profile)
	if err != nil {
		t.Fatal(err)
	}
	if len(saved.Errors) != 1 || saved.Errors[0].Line != 2 {
		t.Fatalf("errors = %+v", saved.Errors)
	}
	if err := rs.ActivateProfile(work.Profile.ID); err != nil {
		t.Fatal(err)
	}
	view, _ = rs.Routing()
	if last := view.Rules[len(view.Rules)-1]; last.Type != "MATCH" || last.Target != "DIRECT" || view.Rules[1].Target != "REJECT" {
		t.Fatalf("rules after switching profile = %+v", view.Rules)
	}
}

func TestRuleKeyword(t *testing.T) {
	for api, want := range map[string]string{"RuleSet": "RULE-SET", "GeoIP": "GEOIP", "Match": "MATCH", "Future": "FUTURE"} {
		if got := ruleKeyword(api); got != want {
			t.Errorf("%s → %s, want %s", api, got, want)
		}
	}
}
