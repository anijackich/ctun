package config

import (
	"cmp"
	"fmt"
	"maps"
	"strings"
)

type Profile struct {
	Proxies        []map[string]any
	ProxyGroups    []map[string]any
	ProxyProviders map[string]any
	RuleProviders  map[string]any
	Rules          []string
}

type Group struct {
	Name    string
	Type    string
	Proxies []string
}

const (
	PrimaryGroup = "PROXY"
	AutoGroup    = "AUTO"

	TestURL = "https://www.gstatic.com/generate_204"
	lanRule = "GEOIP,LAN,DIRECT,no-resolve"
)

type Routing struct {
	Rules     []string
	Providers map[string]any

	Final string

	ProxyLAN bool
}

func ownRouting(p *Profile) ([]map[string]any, []string, map[string]any) {
	if len(p.ProxyGroups) > 0 {
		return p.ProxyGroups, p.Rules, p.RuleProviders
	}
	names := proxyNames(p.Proxies)
	groups := []map[string]any{{"name": PrimaryGroup, "type": "select", "proxies": names}}
	if len(names) > 1 {
		groups[0]["proxies"] = append([]string{AutoGroup}, names...)
		groups = append(groups, map[string]any{
			"name": AutoGroup, "type": "url-test", "proxies": names,
			"url": TestURL, "interval": 300, "tolerance": 50,
		})
	}
	return groups, nil, nil
}

func MainGroup(p *Profile) string {
	groups, _, _ := ownRouting(p)
	for _, g := range groups {
		if g["type"] == "select" {
			return fmt.Sprint(g["name"])
		}
	}
	if len(groups) > 0 {
		return fmt.Sprint(groups[0]["name"])
	}
	return PrimaryGroup
}

func Rules(p *Profile, user *Routing) (rules []string, providers map[string]any) {
	_, own, ownProviders := ownRouting(p)
	providers = maps.Clone(ownProviders)
	if providers == nil {
		providers = map[string]any{}
	}
	if user == nil || !user.ProxyLAN {
		rules = append(rules, lanRule)
	}
	final := MainGroup(p)
	if user != nil {

		renamed := map[string]string{}
		for name, set := range user.Providers {
			key := name
			if _, taken := providers[key]; taken {
				key = "ctun-" + name
				renamed[name] = key
			}
			providers[key] = set
		}
		for _, r := range user.Rules {
			for from, to := range renamed {
				r = strings.Replace(r, "RULE-SET,"+from+",", "RULE-SET,"+to+",", 1)
			}
			rules = append(rules, r)
		}
		final = cmp.Or(user.Final, final)
	}
	for _, r := range own {
		if kind, target, ok := strings.Cut(r, ","); ok && strings.EqualFold(strings.TrimSpace(kind), "MATCH") {
			if user == nil {
				final = strings.TrimSpace(target)
			}
			continue
		}
		rules = append(rules, r)
	}
	return append(rules, "MATCH,"+final), providers
}

func Groups(p *Profile) []Group {
	raw, _, _ := ownRouting(p)
	groups := make([]Group, 0, len(raw))
	for _, g := range raw {
		name, _ := g["name"].(string)
		kind, _ := g["type"].(string)
		groups = append(groups, Group{Name: name, Type: kind, Proxies: stringList(g["proxies"])})
	}
	return groups
}

func proxyNames(proxies []map[string]any) []string {
	names := make([]string, 0, len(proxies))
	for _, p := range proxies {
		if name, ok := p["name"].(string); ok {
			names = append(names, name)
		}
	}
	return names
}

func stringList(v any) []string {
	switch list := v.(type) {
	case []string:
		return list
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
