package sysnet

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type gnomeProxy struct {
	run runner
}

func newProxyBackend(run runner) backend[Proxy] {
	return gnomeProxy{run: run}
}

type gsetting struct{ schema, key string }

var gnomeKeys = []gsetting{
	{"org.gnome.system.proxy", "mode"},
	{"org.gnome.system.proxy", "ignore-hosts"},
	{"org.gnome.system.proxy.http", "host"},
	{"org.gnome.system.proxy.http", "port"},
	{"org.gnome.system.proxy.https", "host"},
	{"org.gnome.system.proxy.https", "port"},
	{"org.gnome.system.proxy.socks", "host"},
	{"org.gnome.system.proxy.socks", "port"},
}

func (b gnomeProxy) snapshot(ctx context.Context) (json.RawMessage, error) {
	values := map[string]string{}
	for _, s := range gnomeKeys {
		out, err := b.run(ctx, "gsettings", "get", s.schema, s.key)
		if err != nil {
			return nil, err
		}
		values[s.schema+" "+s.key] = strings.TrimSpace(out)
	}
	data, err := json.Marshal(values)
	if err != nil {
		return nil, fmt.Errorf("encode snapshot: %w", err)
	}
	return data, nil
}

func (b gnomeProxy) apply(ctx context.Context, proxy Proxy) error {
	quoted := make([]string, len(proxy.Bypass))
	for i, h := range proxy.Bypass {
		quoted[i] = strconv.Quote(h)
	}
	values := map[gsetting]string{
		{"org.gnome.system.proxy", "ignore-hosts"}: "[" + strings.Join(quoted, ", ") + "]",
	}
	for _, schema := range []string{"http", "https", "socks"} {
		values[gsetting{"org.gnome.system.proxy." + schema, "host"}] = proxy.Host
		values[gsetting{"org.gnome.system.proxy." + schema, "port"}] = strconv.Itoa(proxy.Port)
	}
	for s, v := range values {
		if _, err := b.run(ctx, "gsettings", "set", s.schema, s.key, v); err != nil {
			return err
		}
	}
	_, err := b.run(ctx, "gsettings", "set", "org.gnome.system.proxy", "mode", "manual")
	return err
}

func (b gnomeProxy) restore(ctx context.Context, snapshot json.RawMessage) error {
	var values map[string]string
	if err := json.Unmarshal(snapshot, &values); err != nil {
		return fmt.Errorf("decode snapshot: %w", err)
	}
	for _, s := range gnomeKeys {
		if v, ok := values[s.schema+" "+s.key]; ok {
			if _, err := b.run(ctx, "gsettings", "set", s.schema, s.key, v); err != nil {
				return err
			}
		}
	}
	return nil
}
