package sysnet

import (
	"context"
	"encoding/json"
	"fmt"
)

type darwinDNS struct {
	run runner
}

func newDNSBackend(run runner) backend[[]string] {
	return darwinDNS{run: run}
}

type dnsState struct {
	Name    string   `json:"name"`
	Servers []string `json:"servers"`
}

func (b darwinDNS) snapshot(ctx context.Context) (json.RawMessage, error) {
	services, err := networkServices(ctx, b.run)
	if err != nil {
		return nil, err
	}
	states := make([]dnsState, 0, len(services))
	for _, name := range services {
		out, err := b.run(ctx, "networksetup", "-getdnsservers", name)
		if err != nil {
			return nil, err
		}
		states = append(states, dnsState{Name: name, Servers: listLines(out)})
	}
	data, err := json.Marshal(states)
	if err != nil {
		return nil, fmt.Errorf("encode snapshot: %w", err)
	}
	return data, nil
}

func (b darwinDNS) apply(ctx context.Context, servers []string) error {
	services, err := networkServices(ctx, b.run)
	if err != nil {
		return err
	}
	for _, name := range services {
		if err := b.set(ctx, name, servers); err != nil {
			return err
		}
	}
	return nil
}

func (b darwinDNS) restore(ctx context.Context, snapshot json.RawMessage) error {
	var states []dnsState
	if err := json.Unmarshal(snapshot, &states); err != nil {
		return fmt.Errorf("decode snapshot: %w", err)
	}
	for _, st := range states {
		if err := b.set(ctx, st.Name, st.Servers); err != nil {
			return err
		}
	}
	return nil
}

func (b darwinDNS) set(ctx context.Context, service string, servers []string) error {
	if len(servers) == 0 {
		servers = []string{"Empty"}
	}
	_, err := b.run(ctx, "networksetup", append([]string{"-setdnsservers", service}, servers...)...)
	return err
}
