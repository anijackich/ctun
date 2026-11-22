package app

import (
	"cmp"
	"context"
	"log/slog"
	"slices"
	"strings"
	"time"

	"ctun/internal/config"
	"ctun/internal/core"
)

const ruleSetUpdateTimeout = time.Minute

type RuleView struct {
	Index    int    `json:"index"`
	Type     string `json:"type"`
	Payload  string `json:"payload"`
	Target   string `json:"target"`
	Hits     uint64 `json:"hits"`
	LastHit  int64  `json:"lastHit"`
	Disabled bool   `json:"disabled"`
}

type RuleSetView struct {
	Name      string `json:"name"`
	Behavior  string `json:"behavior"`
	Format    string `json:"format"`
	Source    string `json:"source"`
	Rules     int    `json:"rules"`
	UpdatedAt int64  `json:"updatedAt"`
}

type RoutingView struct {
	Live     bool          `json:"live"`
	Rules    []RuleView    `json:"rules"`
	RuleSets []RuleSetView `json:"ruleSets"`
}

func (s *RoutingService) Routing() (RoutingView, error) {
	if client, _ := s.core.running(); client != nil {
		return liveRouting(client)
	}
	id := s.profiles.ActiveID()
	if id == "" {
		return RoutingView{Rules: []RuleView{}, RuleSets: []RuleSetView{}}, nil
	}
	layer, err := s.profiles.Layer(id)
	if err != nil {
		return RoutingView{}, err
	}
	rules, providers := config.Rules(layer, userRouting(s.routes.Active(), layer))
	return configuredRouting(rules, providers), nil
}

func (s *RoutingService) SetRuleDisabled(index int, disabled bool) error {
	return s.core.withClient(func(ctx context.Context, c *core.Client) error {
		return c.SetRuleDisabled(ctx, index, disabled)
	})
}

func (s *RoutingService) UpdateRuleSet(name string) error {
	client, _ := s.core.running()
	if client == nil {
		return errNotRunning
	}
	ctx, cancel := context.WithTimeout(context.Background(), ruleSetUpdateTimeout)
	defer cancel()
	if err := client.UpdateRuleProvider(ctx, name); err != nil {
		return err
	}
	slog.Info("rule set updated", "name", name)
	return nil
}

func liveRouting(client *core.Client) (RoutingView, error) {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
	rules, err := client.Rules(ctx)
	if err != nil {
		return RoutingView{}, err
	}
	providers, err := client.RuleProviders(ctx)
	if err != nil {
		return RoutingView{}, err
	}
	view := RoutingView{Live: true, Rules: make([]RuleView, len(rules)), RuleSets: make([]RuleSetView, 0, len(providers))}
	for i, r := range rules {
		view.Rules[i] = RuleView{Index: r.Index, Type: ruleKeyword(r.Type), Payload: r.Payload, Target: r.Proxy}
		if e := r.Extra; e != nil {
			view.Rules[i].Hits, view.Rules[i].Disabled = e.HitCount, e.Disabled
			if !e.HitAt.IsZero() {
				view.Rules[i].LastHit = e.HitAt.UnixMilli()
			}
		}
	}
	for _, p := range providers {
		set := RuleSetView{
			Name:     p.Name,
			Behavior: strings.ToLower(p.Behavior),
			Format:   strings.ToLower(strings.TrimSuffix(p.Format, "Rule")),
			Source:   strings.ToLower(p.VehicleType),
			Rules:    p.RuleCount,
		}
		if !p.UpdatedAt.IsZero() {
			set.UpdatedAt = p.UpdatedAt.UnixMilli()
		}
		view.RuleSets = append(view.RuleSets, set)
	}
	slices.SortFunc(view.RuleSets, func(a, b RuleSetView) int { return cmp.Compare(a.Name, b.Name) })
	return view, nil
}

func configuredRouting(rules []string, providers map[string]any) RoutingView {
	view := RoutingView{Rules: make([]RuleView, len(rules)), RuleSets: make([]RuleSetView, 0, len(providers))}
	for i, raw := range rules {
		view.Rules[i] = parseRule(i, raw)
	}
	for name, raw := range providers {
		p, _ := raw.(map[string]any)
		view.RuleSets = append(view.RuleSets, RuleSetView{
			Name:     name,
			Behavior: strings.ToLower(str(p["behavior"])),
			Format:   strings.ToLower(cmp.Or(str(p["format"]), "yaml")),
			Source:   strings.ToLower(str(p["type"])),
		})
	}
	slices.SortFunc(view.RuleSets, func(a, b RuleSetView) int { return cmp.Compare(a.Name, b.Name) })
	return view
}

var ruleParams = []string{"no-resolve", "src"}

func parseRule(index int, raw string) RuleView {
	parts := strings.Split(raw, ",")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	rule := RuleView{Index: index, Type: strings.ToUpper(parts[0])}
	rest := parts[1:]
	for len(rest) > 1 && slices.Contains(ruleParams, rest[len(rest)-1]) {
		rest = rest[:len(rest)-1]
	}
	if len(rest) == 0 {
		return rule
	}
	rule.Target = rest[len(rest)-1]
	rule.Payload = strings.Join(rest[:len(rest)-1], ",")
	return rule
}

var ruleKeywords = map[string]string{
	"Domain": "DOMAIN", "DomainSuffix": "DOMAIN-SUFFIX", "DomainKeyword": "DOMAIN-KEYWORD",
	"DomainRegex": "DOMAIN-REGEX", "DomainWildcard": "DOMAIN-WILDCARD", "GeoSite": "GEOSITE",
	"GeoIP": "GEOIP", "SrcGeoIP": "SRC-GEOIP", "IPASN": "IP-ASN", "SrcIPASN": "SRC-IP-ASN",
	"IPCIDR": "IP-CIDR", "SrcIPCIDR": "SRC-IP-CIDR", "IPSuffix": "IP-SUFFIX",
	"SrcIPSuffix": "SRC-IP-SUFFIX", "SrcPort": "SRC-PORT", "DstPort": "DST-PORT",
	"InPort": "IN-PORT", "InUser": "IN-USER", "InName": "IN-NAME", "InType": "IN-TYPE",
	"ProcessName": "PROCESS-NAME", "ProcessPath": "PROCESS-PATH",
	"ProcessNameRegex": "PROCESS-NAME-REGEX", "ProcessPathRegex": "PROCESS-PATH-REGEX",
	"ProcessNameWildcard": "PROCESS-NAME-WILDCARD", "ProcessPathWildcard": "PROCESS-PATH-WILDCARD",
	"RematchName": "REMATCH-NAME", "Match": "MATCH", "RuleSet": "RULE-SET", "Network": "NETWORK",
	"DSCP": "DSCP", "Uid": "UID", "SubRules": "SUB-RULE", "AND": "AND", "OR": "OR", "NOT": "NOT",
}

func ruleKeyword(apiType string) string {
	if keyword, ok := ruleKeywords[apiType]; ok {
		return keyword
	}
	return strings.ToUpper(apiType)
}

func str(v any) string {
	s, _ := v.(string)
	return s
}
