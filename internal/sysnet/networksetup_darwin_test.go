package sysnet

import (
	"context"
	"slices"
	"strings"
	"testing"
)

type fakeNetworksetup struct {
	proxies map[string]string
	bypass  map[string][]string
	dns     map[string][]string
	calls   []string
}

func (f *fakeNetworksetup) run(_ context.Context, _ string, args ...string) (string, error) {
	f.calls = append(f.calls, strings.Join(args, " "))
	cmd := args[0]
	switch {
	case cmd == "-listallnetworkservices":
		return "An asterisk (*) denotes that a network service is disabled.\nWi-Fi\nUSB LAN\n*Old VPN\n", nil
	case cmd == "-getproxybypassdomains":
		return lines(f.bypass[args[1]], "There aren't any bypass domains set on "+args[1]+"."), nil
	case cmd == "-getdnsservers":
		return lines(f.dns[args[1]], "There aren't any DNS Servers set on "+args[1]+"."), nil
	case cmd == "-setdnsservers":
		if args[2] == "Empty" {
			delete(f.dns, args[1])
		} else {
			f.dns[args[1]] = args[2:]
		}
	case strings.HasPrefix(cmd, "-get"):
		return f.proxies[cmd+" "+args[1]], nil
	}
	return "", nil
}

func lines(values []string, empty string) string {
	if len(values) == 0 {
		return empty + "\n"
	}
	return strings.Join(values, "\n") + "\n"
}

func TestProxySnapshotApplyRestore(t *testing.T) {
	f := &fakeNetworksetup{
		proxies: map[string]string{"-getwebproxy Wi-Fi": "Enabled: No\nServer: 127.0.0.1\nPort: 10808\n"},
		bypass:  map[string][]string{"Wi-Fi": {"*.local", "169.254/16"}},
	}
	b := darwinProxy{run: f.run}
	ctx := t.Context()

	snapshot, err := b.snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.apply(ctx, Proxy{Host: "127.0.0.1", Port: 7890, Bypass: LocalBypass}); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"-setwebproxy Wi-Fi 127.0.0.1 7890", "-setsocksfirewallproxy USB LAN 127.0.0.1 7890"} {
		if !slices.Contains(f.calls, want) {
			t.Errorf("missing call %q", want)
		}
	}
	if slices.ContainsFunc(f.calls, func(c string) bool { return strings.Contains(c, "Old VPN") }) {
		t.Error("a disabled service was touched")
	}

	f.calls = nil
	if err := b.restore(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"-setwebproxy Wi-Fi 127.0.0.1 10808",
		"-setwebproxystate Wi-Fi off",
		"-setsecurewebproxystate USB LAN off",
		"-setproxybypassdomains Wi-Fi *.local 169.254/16",
		"-setproxybypassdomains USB LAN Empty",
	} {
		if !slices.Contains(f.calls, want) {
			t.Errorf("missing restore call %q in %q", want, f.calls)
		}
	}
}

func TestDNSSnapshotApplyRestore(t *testing.T) {
	f := &fakeNetworksetup{dns: map[string][]string{"USB LAN": {"192.0.2.53", "192.0.2.54"}}}
	b := darwinDNS{run: f.run}
	ctx := t.Context()

	snapshot, err := b.snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := b.apply(ctx, []string{"1.1.1.1"}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(f.dns["Wi-Fi"], []string{"1.1.1.1"}) || !slices.Equal(f.dns["USB LAN"], []string{"1.1.1.1"}) {
		t.Fatalf("dns after apply = %v", f.dns)
	}
	if err := b.restore(ctx, snapshot); err != nil {
		t.Fatal(err)
	}
	if _, dhcp := f.dns["Wi-Fi"]; dhcp || !slices.Equal(f.dns["USB LAN"], []string{"192.0.2.53", "192.0.2.54"}) {
		t.Fatalf("dns after restore = %v", f.dns)
	}
}
