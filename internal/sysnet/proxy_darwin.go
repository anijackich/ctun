package sysnet

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type darwinProxy struct {
	run runner
}

func newProxyBackend(run runner) backend[Proxy] {
	return darwinProxy{run: run}
}

type endpoint struct {
	Enabled bool   `json:"enabled"`
	Host    string `json:"host"`
	Port    int    `json:"port"`
}

type protocol struct {
	get, set, state string
}

var protocols = map[string]protocol{
	"web":    {"-getwebproxy", "-setwebproxy", "-setwebproxystate"},
	"secure": {"-getsecurewebproxy", "-setsecurewebproxy", "-setsecurewebproxystate"},
	"socks":  {"-getsocksfirewallproxy", "-setsocksfirewallproxy", "-setsocksfirewallproxystate"},
}

type proxyState struct {
	Name      string              `json:"name"`
	Endpoints map[string]endpoint `json:"endpoints"`
	Bypass    []string            `json:"bypass"`
}

func (b darwinProxy) snapshot(ctx context.Context) (json.RawMessage, error) {
	services, err := networkServices(ctx, b.run)
	if err != nil {
		return nil, err
	}
	states := make([]proxyState, 0, len(services))
	for _, name := range services {
		st := proxyState{Name: name, Endpoints: map[string]endpoint{}}
		for key, p := range protocols {
			out, err := b.run(ctx, "networksetup", p.get, name)
			if err != nil {
				return nil, err
			}
			st.Endpoints[key] = parseEndpoint(out)
		}
		out, err := b.run(ctx, "networksetup", "-getproxybypassdomains", name)
		if err != nil {
			return nil, err
		}
		st.Bypass = listLines(out)
		states = append(states, st)
	}
	data, err := json.Marshal(states)
	if err != nil {
		return nil, fmt.Errorf("encode snapshot: %w", err)
	}
	return data, nil
}

func (b darwinProxy) apply(ctx context.Context, proxy Proxy) error {
	services, err := networkServices(ctx, b.run)
	if err != nil {
		return err
	}
	for _, name := range services {
		for _, p := range protocols {
			if _, err := b.run(ctx, "networksetup", p.set, name, proxy.Host, strconv.Itoa(proxy.Port)); err != nil {
				return err
			}
		}
		if _, err := b.run(ctx, "networksetup", append([]string{"-setproxybypassdomains", name}, proxy.Bypass...)...); err != nil {
			return err
		}
	}
	return nil
}

func (b darwinProxy) restore(ctx context.Context, snapshot json.RawMessage) error {
	var states []proxyState
	if err := json.Unmarshal(snapshot, &states); err != nil {
		return fmt.Errorf("decode snapshot: %w", err)
	}
	for _, st := range states {
		for key, p := range protocols {
			e := st.Endpoints[key]
			// Setting an address switches the proxy on, so its state goes last.
			if e.Host != "" {
				if _, err := b.run(ctx, "networksetup", p.set, st.Name, e.Host, strconv.Itoa(e.Port)); err != nil {
					return err
				}
			}
			if _, err := b.run(ctx, "networksetup", p.state, st.Name, onOff(e.Enabled)); err != nil {
				return err
			}
		}
		bypass := st.Bypass
		if len(bypass) == 0 {
			bypass = []string{"Empty"}
		}
		if _, err := b.run(ctx, "networksetup", append([]string{"-setproxybypassdomains", st.Name}, bypass...)...); err != nil {
			return err
		}
	}
	return nil
}

func parseEndpoint(out string) endpoint {
	var e endpoint
	for line := range strings.Lines(out) {
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		switch strings.TrimSpace(key) {
		case "Enabled":
			e.Enabled = value == "Yes"
		case "Server":
			e.Host = value
		case "Port":
			e.Port, _ = strconv.Atoi(value)
		}
	}
	return e
}

func onOff(on bool) string {
	if on {
		return "on"
	}
	return "off"
}
