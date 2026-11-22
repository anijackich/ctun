package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ctun/internal/core"
	"ctun/internal/profile"
)

func TestBuildGroupsStaticAndLive(t *testing.T) {
	layer, err := profile.Parse([]byte(testSubscription))
	if err != nil {
		t.Fatal(err)
	}

	static := buildGroups(layer, map[string]string{"PROXY": "Finland 🇫🇮"}, nil, nil)
	if len(static) != 2 || static[0].Name != "PROXY" || static[0].Now != "Finland 🇫🇮" {
		t.Fatalf("static = %+v", static)
	}
	if static[0].Nodes[0].Name != "AUTO" || static[0].Nodes[0].Protocol != "" || static[0].Nodes[1].Protocol != "SOCKS5" {
		t.Fatalf("static nodes = %+v", static[0].Nodes)
	}
	if static[1].Now != "Germany 🇩🇪" {
		t.Errorf("url-test without live data defaults to the first member, got %q", static[1].Now)
	}

	var live map[string]core.ProxyState
	if err := json.Unmarshal([]byte(`{
		"PROXY": {"name":"PROXY","type":"Selector","now":"AUTO","all":["AUTO","Germany 🇩🇪","Finland 🇫🇮"]},
		"AUTO": {"name":"AUTO","type":"URLTest","now":"Finland 🇫🇮","all":["Germany 🇩🇪","Finland 🇫🇮"],"history":[{"delay":40}]},
		"Germany 🇩🇪": {"name":"Germany 🇩🇪","type":"Socks5","history":[{"delay":90},{"delay":0}]},
		"Finland 🇫🇮": {"name":"Finland 🇫🇮","type":"Socks5","history":[{"delay":40}]}
	}`), &live); err != nil {
		t.Fatal(err)
	}
	groups := buildGroups(layer, map[string]string{"PROXY": "Finland 🇫🇮"}, live, nil)
	if groups[0].Now != "AUTO" || groups[1].Now != "Finland 🇫🇮" {
		t.Fatalf("live selection must win: %+v", groups)
	}
	if d := groups[0].Nodes[1].Delay; d == nil || *d != 0 {
		t.Errorf("Germany delay = %v, want last result 0 (failed)", d)
	}
	if d := groups[0].Nodes[0].Delay; d == nil || *d != 40 {
		t.Errorf("AUTO delay = %v, want 40", d)
	}
}

func TestBuildGroupsOfflineUsesMeasuredDelays(t *testing.T) {
	layer, err := profile.Parse([]byte(testSubscription))
	if err != nil {
		t.Fatal(err)
	}
	groups := buildGroups(layer, nil, nil, map[string]int{"Germany 🇩🇪": 0, "Finland 🇫🇮": 55})
	auto := groups[1]
	if auto.Now != "Finland 🇫🇮" {
		t.Errorf("offline AUTO must pick the fastest measured server, got %q", auto.Now)
	}
	if d := auto.Nodes[0].Delay; d == nil || *d != 0 {
		t.Errorf("failed measurement must be reported as 0, got %v", d)
	}
	if d := auto.Nodes[1].Delay; d == nil || *d != 55 {
		t.Errorf("Finland delay = %v, want 55", d)
	}
}

func TestImportConfigCreatesLocalProfile(t *testing.T) {
	s := newTestService(t, proxyMode, false)
	ps := NewProfileService(s.profiles, s)
	link := "vless://00000000-0000-0000-0000-000000000000@nl.example.com:443?type=ws&security=tls&sni=nl.example.com#Netherlands%20%F0%9F%87%B3%F0%9F%87%B1"
	view, err := ps.Import("  " + link + "\n")
	if err != nil {
		t.Fatal(err)
	}
	if !view.Local || view.Name != "Netherlands 🇳🇱" || !view.Active {
		t.Fatalf("view = %+v", view)
	}
	if err := ps.Refresh(view.ID); err == nil {
		t.Error("local configs must not refresh")
	}
	if _, err := ps.SubscriptionURL(view.ID); err == nil {
		t.Error("local configs have no link to copy")
	}
	if _, err := ps.Import("not a config"); err == nil {
		t.Error("garbage must be rejected")
	}
}

func TestEditProfile(t *testing.T) {
	s := newTestService(t, proxyMode, true)
	ps := NewProfileService(s.profiles, s)

	view, err := ps.Edit(EditProfileRequest{ID: "p1", Name: " Renamed ", UpdateIntervalHours: 6})
	if err != nil {
		t.Fatal(err)
	}
	if view.Name != "Renamed" || view.UpdateIntervalHours != 6 || view.UserAgent != profile.DefaultUserAgent {
		t.Fatalf("view = %+v", view)
	}
	if p, _ := s.profiles.Get("p1"); p.URL != "https://sub.example.com/x" {
		t.Fatal("an empty link must keep the current one")
	}

	served := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		served++
		if r.URL.Path == "/broken" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(testSubscription))
	}))
	defer srv.Close()
	if _, err := ps.Edit(EditProfileRequest{ID: "p1", Name: "Renamed", URL: srv.URL + "/broken"}); err == nil {
		t.Fatal("a link that does not download must be rejected")
	}
	if p, _ := s.profiles.Get("p1"); p.URL != "https://sub.example.com/x" {
		t.Fatal("a failed edit must not change the subscription")
	}
	if _, err := ps.Edit(EditProfileRequest{ID: "p1", Name: "Renamed", URL: srv.URL + "/new"}); err != nil {
		t.Fatal(err)
	}
	if p, _ := s.profiles.Get("p1"); p.URL != srv.URL+"/new" || p.Name != "Renamed" || served != 2 {
		t.Fatalf("profile = %+v, served = %d", p, served)
	}
	if _, err := ps.Edit(EditProfileRequest{ID: "p1", Name: "  "}); err == nil {
		t.Fatal("an empty name must be rejected")
	}

	local, err := ps.Import("vless://00000000-0000-0000-0000-000000000000@nl.example.com:443?type=tcp#NL")
	if err != nil {
		t.Fatal(err)
	}
	if view, err := ps.Edit(EditProfileRequest{ID: local.ID, Name: "Mine", URL: srv.URL, UserAgent: "x"}); err != nil || view.Name != "Mine" || view.UserAgent != "" {
		t.Fatalf("local config edit = %+v, %v", view, err)
	}
}
