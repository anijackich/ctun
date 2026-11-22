package routing

import (
	"slices"
	"strings"
)

type Target string

const (
	Block  Target = "block"
	Direct Target = "direct"
	Proxy  Target = "proxy"
)

var Targets = []Target{Block, Direct, Proxy}

func (t Target) valid() bool {
	return slices.Contains(Targets, t)
}

type List struct {
	Target  Target   `json:"target"`
	Entries []string `json:"entries"`
}

type Profile struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Lists []List `json:"lists"`

	ProxyLAN bool   `json:"proxyLan"`
	Final    Target `json:"final"`
}

const defaultID = "default"

func Default() Profile {
	return Profile{ID: defaultID, Name: "Default", Final: Proxy}.normalize()
}

func (p Profile) normalize() Profile {
	lists := make([]List, 0, len(Targets))
	seen := map[Target]bool{}
	for _, l := range p.Lists {
		if !l.Target.valid() || seen[l.Target] {
			continue
		}
		seen[l.Target] = true
		lists = append(lists, List{Target: l.Target, Entries: trimEntries(l.Entries)})
	}
	for _, t := range Targets {
		if !seen[t] {
			lists = append(lists, List{Target: t, Entries: []string{}})
		}
	}
	p.Lists = lists
	p.Name = strings.TrimSpace(p.Name)
	if !p.Final.valid() {
		p.Final = Proxy
	}
	return p
}

func trimEntries(entries []string) []string {
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, strings.TrimRight(e, " \t\r"))
	}
	for len(out) > 0 && strings.TrimSpace(out[len(out)-1]) == "" {
		out = out[:len(out)-1]
	}
	return out
}
