package app

import (
	"context"
	"log/slog"
	"time"

	"ctun/internal/config"
	"ctun/internal/latency"
)

const latencyBudget = 2 * time.Minute

// TestLatency measures every server of a group and streams each result as an event. It goes
// through the core when it runs this subscription (the app's own traffic may be tunnelled
// then), otherwise directly, so no connection is needed.
func (s *ProfileService) TestLatency(id, group string) ([]GroupView, error) {
	layer, err := s.store.Layer(id)
	if err != nil {
		return nil, err
	}
	s.measure(id, layer, groupServers(layer, group))
	return s.Groups(id)
}

func (s *ProfileService) testActive() {
	if client, _ := s.core.running(); client == nil {
		return
	}
	id := s.store.ActiveID()
	layer, err := s.store.Layer(id)
	if err != nil {
		return
	}
	groups, err := s.Groups(id)
	if err != nil {
		return
	}
	_, server := activeServer(groups)
	if !proxyNames(layer)[server.Name] {
		return
	}
	slog.Info("testing the active server", "server", server.Name)
	s.measure(id, layer, []string{server.Name})
}

func (s *ProfileService) measure(id string, layer *config.Profile, names []string) {
	var test latency.Tester
	if client, _ := s.core.running(); client != nil && id == s.store.ActiveID() {
		test = func(ctx context.Context, name string) (int, error) {
			return client.ProxyDelay(ctx, name, config.TestURL, latency.Timeout)
		}
	} else {
		test = latency.Direct(layer.Proxies, s.core.settings.Get().DNS.Servers)
	}
	ctx, cancel := context.WithTimeout(context.Background(), latencyBudget)
	defer cancel()
	latency.Run(ctx, names, test, func(name string, delay int) {
		s.delaysMu.Lock()
		if s.delays[id] == nil {
			s.delays[id] = map[string]int{}
		}
		s.delays[id][name] = delay
		s.delaysMu.Unlock()
		emit(EventLatency, LatencyEvent{ProfileID: id, Name: name, Delay: delay})
	})
}

func groupServers(layer *config.Profile, group string) []string {
	known := proxyNames(layer)
	var names []string
	for _, g := range config.Groups(layer) {
		if g.Name != group {
			continue
		}
		for _, name := range g.Proxies {
			if known[name] {
				names = append(names, name)
			}
		}
	}
	return names
}

func proxyNames(layer *config.Profile) map[string]bool {
	names := make(map[string]bool, len(layer.Proxies))
	for _, proxy := range layer.Proxies {
		if name, ok := proxy["name"].(string); ok {
			names[name] = true
		}
	}
	return names
}
