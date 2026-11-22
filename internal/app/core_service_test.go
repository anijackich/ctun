package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ctun/internal/privilege"
	"ctun/internal/profile"
	"ctun/internal/routing"
	"ctun/internal/settings"
)

const testSubscription = `
proxies:
  - {name: "Germany 🇩🇪", type: socks5, server: 127.0.0.1, port: 1080}
  - {name: "Finland 🇫🇮", type: socks5, server: 127.0.0.1, port: 1081}
`

func newTestService(t *testing.T, mutate func(*settings.Settings), withProfile bool) *CoreService {
	t.Helper()
	dir := t.TempDir()
	paths := Paths{Data: dir, Runtime: dir, Profiles: filepath.Join(dir, "profiles"), Settings: filepath.Join(dir, "settings.json")}
	store, err := settings.Open(paths.Settings)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Update(mutate); err != nil {
		t.Fatal(err)
	}
	profiles, err := profile.OpenStore(paths.Profiles)
	if err != nil {
		t.Fatal(err)
	}
	if withProfile {
		p := profile.Profile{ID: "p1", Name: "Test", URL: "https://sub.example.com/x", Selected: map[string]string{"PROXY": "Finland 🇫🇮"}}
		if err := profiles.Put(p, []byte(testSubscription)); err != nil {
			t.Fatal(err)
		}
	}
	routes, err := routing.Open(filepath.Join(dir, "routing.json"))
	if err != nil {
		t.Fatal(err)
	}
	return NewCoreService(paths, store, profiles, routes)
}

func proxyMode(st *settings.Settings) { st.TunnelMode = settings.TunnelProxy }

func TestStagedConfigUsesActiveProfileAndPrivateFile(t *testing.T) {
	s := newTestService(t, proxyMode, true)
	creds, err := newRuntimeCreds(s.settings.Get())
	if err != nil {
		t.Fatal(err)
	}
	cfg, selected, err := s.buildConfig(s.settings.Get(), creds)
	if err != nil {
		t.Fatal(err)
	}
	path, err := s.local.Stage(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if selected["PROXY"] != "Finland 🇫🇮" {
		t.Errorf("selections = %v", selected)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("config mode = %o, want 600", perm)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{
		"secret: " + creds.secret,
		"external-controller: " + creds.controllerAddr,
		"- ctun:" + creds.password,
		"name: PROXY",
		"server: 127.0.0.1",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("config lacks %q", want)
		}
	}
}

func TestRuntimeCredsAreFreshAndLoopbackOnly(t *testing.T) {
	opts := settings.Default()
	opts.TunnelMode = settings.TunnelProxy
	first, err := newRuntimeCreds(opts)
	if err != nil {
		t.Fatal(err)
	}
	second, err := newRuntimeCreds(opts)
	if err != nil {
		t.Fatal(err)
	}
	if first.secret == second.secret || first.password == second.password {
		t.Error("secrets must change on every launch")
	}
	if !strings.HasPrefix(first.controllerAddr, "127.0.0.1:") {
		t.Errorf("controller must be loopback-only, got %s", first.controllerAddr)
	}
	if got := first.proxyURL(); !strings.HasPrefix(got, "http://ctun:") || !strings.HasSuffix(got, "@127.0.0.1:7890") {
		t.Errorf("proxy URL = %q", got)
	}

	opts.PortAuth = false
	anon, err := newRuntimeCreds(opts)
	if err != nil {
		t.Fatal(err)
	}
	if anon.password != "" || anon.proxyURL() != "http://127.0.0.1:7890" {
		t.Errorf("anonymous creds = %+v", anon)
	}
}

func TestStartWithoutProfileReportsCode(t *testing.T) {
	s := newTestService(t, proxyMode, false)
	if err := s.Start(); err == nil {
		t.Fatal("expected error")
	}
	if got := s.State(); got.ErrorCode != CodeNoProfile && got.ErrorCode != CodeCoreMissing {
		t.Fatalf("state = %+v", got)
	}
}

func TestTUNConfigOpensNoPortAndNeedsHelper(t *testing.T) {
	s := newTestService(t, func(st *settings.Settings) { st.TunnelMode = settings.TunnelTUN }, true)
	creds, err := newRuntimeCreds(s.settings.Get())
	if err != nil {
		t.Fatal(err)
	}
	if creds.port != 0 || creds.password != "" || creds.proxyURL() != "" {
		t.Fatalf("TUN must not open a local proxy: %+v", creds)
	}
	cfg, _, err := s.buildConfig(s.settings.Get(), creds)
	if err != nil {
		t.Fatal(err)
	}
	if text := string(cfg); !strings.Contains(text, "tun:") || strings.Contains(text, "mixed-port") {
		t.Fatalf("unexpected TUN config:\n%s", text)
	}

	if privilege.Check(context.Background()) == privilege.StateInstalled {
		t.Skip("helper is installed on this machine")
	}
	if err := s.Start(); err == nil {
		t.Fatal("expected error")
	}
	if got := s.State().ErrorCode; got != CodeHelperMissing && got != CodeHelperOutdated && got != CodeTunUnavailable {
		t.Fatalf("error code = %q", got)
	}
}
