package app

import (
	"runtime"
	"testing"

	"ctun/internal/core"
	"ctun/internal/profile"
	"ctun/internal/settings"
)

func TestTrayStatusLine(t *testing.T) {
	en := trayTexts["en"]
	cases := []struct {
		state  StatusEvent
		server string
		want   string
	}{
		{StatusEvent{Status: core.StatusRunning}, "Finland 🇫🇮", "Connected · Finland 🇫🇮"},
		{StatusEvent{Status: core.StatusRunning}, "", "Connected"},
		{StatusEvent{Status: core.StatusStarting}, "", "Connecting…"},
		{StatusEvent{Status: core.StatusStopped, Error: "boom"}, "", "Couldn't connect"},
		{StatusEvent{Status: core.StatusStopped}, "Finland 🇫🇮", "Disconnected"},
	}
	for _, tc := range cases {
		st := traySnapshot{text: en, state: tc.state, server: tc.server}
		if got := st.statusLine(); got != tc.want {
			t.Errorf("%+v → %q, want %q", tc.state, got, tc.want)
		}
	}
}

func TestTrayTextFallsBackToEnglish(t *testing.T) {
	if trayTextFor("ru").connect != "Подключить" || trayTextFor("de").connect != "Connect" {
		t.Fatal("unexpected tray texts")
	}
}

func TestLanguageParsing(t *testing.T) {
	if got := firstAppleLanguage("(\n    \"ru-RU\",\n    \"en-RU\"\n)\n"); got != "ru" {
		t.Errorf("AppleLanguages → %q", got)
	}
	if got := firstAppleLanguage("(\n    en\n)"); got != "en" {
		t.Errorf("unquoted AppleLanguages → %q", got)
	}
	for tag, want := range map[string]string{"ru_RU.UTF-8": "ru", "en-US": "en", "DE": "de"} {
		if got := languageCode(tag); got != want {
			t.Errorf("languageCode(%q) = %q, want %q", tag, got, want)
		}
	}
}

func TestActivateSwitchesSubscription(t *testing.T) {
	s := newTestService(t, proxyMode, true)
	if err := s.profiles.Put(profile.Profile{ID: "p2", Name: "Second", URL: "https://sub.example.com/y"}, []byte(testSubscription)); err != nil {
		t.Fatal(err)
	}
	ps := NewProfileService(s.profiles, s)
	if err := ps.Activate("p2"); err != nil {
		t.Fatal(err)
	}
	if got := s.profiles.ActiveID(); got != "p2" {
		t.Fatalf("active = %q", got)
	}
	if err := ps.Activate("missing"); err == nil {
		t.Error("unknown subscription must fail")
	}
}

func TestTrayIcon(t *testing.T) {
	delay := func(ms int) *int { return &ms }
	running := StatusEvent{Status: core.StatusRunning}
	cases := []struct {
		st   traySnapshot
		want trayIcon
	}{
		{traySnapshot{state: StatusEvent{Status: core.StatusStopped}, server: "NL 🇳🇱"}, trayIcon{bars: -1}},
		{traySnapshot{state: running}, trayIcon{bars: 4}},
		{traySnapshot{state: running, delay: delay(90)}, trayIcon{bars: 4}},
		{traySnapshot{state: running, delay: delay(250)}, trayIcon{bars: 3}},
		{traySnapshot{state: running, delay: delay(900)}, trayIcon{bars: 1}},
		{traySnapshot{state: running, delay: delay(0)}, trayIcon{bars: 0}},
		{traySnapshot{settings: settings.Settings{TrayIcon: settings.TrayIconLogo}}, trayIcon{logo: true}},
		{traySnapshot{settings: settings.Settings{TrayIcon: settings.TrayIconLogo}, state: running}, trayIcon{logo: true}},
	}
	for _, tc := range cases {
		if got := tc.st.icon(); got != tc.want {
			t.Errorf("%+v → %+v, want %+v", tc.st, got, tc.want)
		}
	}
}

func TestFlagOf(t *testing.T) {
	for name, want := range map[string]string{
		"🇳🇱 Amsterdam": "🇳🇱",
		"Finland 🇫🇮 2": "🇫🇮",
		"Germany":      "",
		"":             "",
	} {
		if got := flagOf(name); got != want {
			t.Errorf("flagOf(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestActiveServerFollowsNestedGroups(t *testing.T) {
	ms := 120
	groups := []GroupView{
		{Name: "AUTO", Type: "url-test", Now: "NL", Nodes: []NodeView{{Name: "NL", Delay: &ms}, {Name: "DE"}}},
		{Name: "PROXY", Type: "select", Now: "AUTO", Nodes: []NodeView{{Name: "AUTO"}, {Name: "NL", Delay: &ms}}},
	}
	primary, server := activeServer(groups)
	if primary.Name != "PROXY" || server.Name != "NL" || server.Delay == nil || *server.Delay != ms {
		t.Fatalf("primary %q, server %+v", primary.Name, server)
	}
	if primary, server := activeServer(nil); primary != nil || server.Name != "" {
		t.Fatal("no groups must give no server")
	}
}

func TestTrayIconFlag(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("the flag icon is macOS-only")
	}
	flag := settings.Settings{TrayIcon: settings.TrayIconFlag}
	running := StatusEvent{Status: core.StatusRunning}
	for server, want := range map[string]string{"NL 🇳🇱": "🇳🇱", "🌐 Auto": genericFlag, "Germany": genericFlag} {
		st := traySnapshot{settings: flag, state: running, server: server}
		if got := st.icon(); got.flag != want {
			t.Errorf("%q → %+v, want %q", server, got, want)
		}
	}
}
