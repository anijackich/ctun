package settings

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func TestOpenMissingFileReturnsDefaults(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "settings.json"))
	if err != nil {
		t.Fatal(err)
	}
	if got := store.Get(); !reflect.DeepEqual(got, Default()) {
		t.Fatalf("got %+v, want defaults", got)
	}
}

func TestUpdatePersistsAndReloads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := store.Update(func(s *Settings) {
		s.Locale = "ru"
		s.RoutingMode = RoutingGlobal
		s.InspectorWidth = 400
	}); err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("file mode = %o, want 600", perm)
	}

	reloaded, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got := reloaded.Get()
	if got.Locale != "ru" || got.RoutingMode != RoutingGlobal || got.InspectorWidth != 400 {
		t.Fatalf("reloaded %+v", got)
	}
}

func TestInvalidValuesAreNormalized(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	raw := `{"tunnelMode":"bogus","routingMode":"x","mixedPort":80,"logLevel":"loud","inspectorWidth":9000,"theme":"sepia"}`
	if err := os.WriteFile(path, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}

	store, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	got, def := store.Get(), Default()
	if got.TunnelMode != def.TunnelMode || got.RoutingMode != def.RoutingMode ||
		got.MixedPort != def.MixedPort || got.LogLevel != def.LogLevel || got.Theme != "" {
		t.Fatalf("not normalized: %+v", got)
	}
	if got.InspectorWidth != maxInspectorWidth {
		t.Errorf("inspector width = %d, want clamped to %d", got.InspectorWidth, maxInspectorWidth)
	}
}

func TestCorruptFileIsAnError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Open(path); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestDNSIsNormalized(t *testing.T) {
	dns := DNS{
		Nameservers: []string{" tls://1.1.1.1:853 ", "dns.google", "tls://1.1.1.1", ""},
		Direct:      []string{"https://dns.google/dns-query"},
		Servers:     []string{"9.9.9.9"},
	}.normalize()
	if !slices.Equal(dns.Nameservers, []string{"tls://1.1.1.1"}) {
		t.Errorf("nameservers = %v", dns.Nameservers)
	}
	if !slices.Equal(dns.Direct, DefaultDNS().Direct) {
		t.Errorf("an emptied list must return to the default, got %v", dns.Direct)
	}
	if dns.SystemServer() != "1.1.1.1" {
		t.Errorf("system server = %q", dns.SystemServer())
	}
	if got := InvalidNameservers([]string{"1.1.1.1", "", "dns.google"}); !slices.Equal(got, []int{3}) {
		t.Errorf("invalid lines = %v", got)
	}
}
