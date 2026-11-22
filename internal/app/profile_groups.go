package app

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"slices"

	"ctun/internal/config"
	"ctun/internal/core"
	"ctun/internal/profile"
)

type NodeView struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Delay    *int   `json:"delay,omitempty"`
}

type GroupView struct {
	Name  string     `json:"name"`
	Type  string     `json:"type"`
	Now   string     `json:"now"`
	Nodes []NodeView `json:"nodes"`
}

func (s *ProfileService) Groups(id string) ([]GroupView, error) {
	p, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	layer, err := s.store.Layer(id)
	if err != nil {
		return nil, err
	}
	var live map[string]core.ProxyState
	if client, _ := s.core.running(); client != nil && id == s.store.ActiveID() {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		live, _ = client.Proxies(ctx)
		cancel()
	}
	s.delaysMu.Lock()
	measured := maps.Clone(s.delays[id])
	s.delaysMu.Unlock()
	return buildGroups(layer, p.Selected, live, measured), nil
}

func (s *ProfileService) Select(id, group, name string) error {
	if _, err := s.store.Update(id, func(p *profile.Profile) {
		if p.Selected == nil {
			p.Selected = map[string]string{}
		}
		p.Selected[group] = name
	}); err != nil {
		return err
	}
	slog.Info("server selected", "group", group, "server", name)
	if id != s.store.ActiveID() {
		return s.Activate(id)
	}
	s.changed()
	err := s.core.withClient(func(ctx context.Context, c *core.Client) error {
		if err := c.SelectProxy(ctx, group, name); err != nil {
			return err
		}
		return c.CloseConnections(ctx)
	})
	switch {
	case errors.Is(err, errNotRunning):
		return nil
	case err != nil:
		return err
	}
	go s.testActive()
	return nil
}

func buildGroups(layer *config.Profile, selected map[string]string, live map[string]core.ProxyState, measured map[string]int) []GroupView {
	protocols := make(map[string]string, len(layer.Proxies))
	for _, proxy := range layer.Proxies {
		if name, ok := proxy["name"].(string); ok {
			protocols[name] = profile.Describe(proxy)
		}
	}
	groups := config.Groups(layer)
	isGroup := make(map[string]bool, len(groups))
	for _, g := range groups {
		isGroup[g.Name] = true
	}
	delayOf := func(name string) *int {
		if d, ok := measured[name]; ok {
			return &d
		}
		if state, ok := live[name]; ok {
			if d, ok := state.LastDelay(); ok {
				return &d
			}
		}
		return nil
	}

	views := make([]GroupView, 0, len(groups))
	for _, g := range groups {
		view := GroupView{Name: g.Name, Type: g.Type}
		members := g.Proxies
		state, isLive := live[g.Name]
		switch {
		case isLive:
			members, view.Now = state.All, state.Now
		case selected[g.Name] != "" && slices.Contains(members, selected[g.Name]):
			view.Now = selected[g.Name]
		case g.Type == "url-test":
			view.Now = fastest(members, measured)
		}
		if view.Now == "" && len(members) > 0 {
			view.Now = members[0]
		}
		for _, name := range members {
			node := NodeView{Name: name, Protocol: protocols[name], Delay: delayOf(name)}
			if node.Protocol == "" && !isGroup[name] {
				node.Protocol = name
			}
			view.Nodes = append(view.Nodes, node)
		}
		views = append(views, view)
	}
	return views
}

func activeServer(groups []GroupView) (*GroupView, NodeView) {
	if len(groups) == 0 {
		return nil, NodeView{}
	}
	primary := &groups[0]
	for i := range groups {
		if groups[i].Type == "select" {
			primary = &groups[i]
			break
		}
	}
	byName := make(map[string]*GroupView, len(groups))
	for i := range groups {
		byName[groups[i].Name] = &groups[i]
	}
	name := primary.Now
	for range groups {
		g, ok := byName[name]
		if !ok || g.Now == "" {
			break
		}
		name = g.Now
	}
	for _, g := range groups {
		for _, node := range g.Nodes {
			if node.Name == name {
				return primary, node
			}
		}
	}
	return primary, NodeView{Name: name}
}

func fastest(members []string, measured map[string]int) string {
	best, bestDelay := "", 0
	for _, name := range members {
		if d := measured[name]; d > 0 && (bestDelay == 0 || d < bestDelay) {
			best, bestDelay = name, d
		}
	}
	return best
}
