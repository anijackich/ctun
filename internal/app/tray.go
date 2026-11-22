package app

import (
	"cmp"
	"errors"
	"log/slog"
	"runtime"
	"sync"
	"unicode"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/wailsapp/wails/v3/pkg/events"

	"ctun/internal/config"
	"ctun/internal/core"
	"ctun/internal/settings"
	"ctun/internal/trayicon"
)

type Tray struct {
	app       *application.App
	window    application.Window
	tray      *application.SystemTray
	menu      *application.Menu
	glyphs    trayGlyphs
	core      *CoreService
	settings  *SettingsService
	profiles  *ProfileService
	privilege *PrivilegeService
	routing   *RoutingService
	dirty     chan struct{}

	mu       sync.Mutex
	shown    trayIcon
	hasShown bool
}

type trayGlyphs struct {
	off  trayicon.Glyph
	on   [trayicon.Bars + 1]trayicon.Glyph
	logo trayicon.Glyph
}

type trayIcon struct {
	logo bool
	flag string
	bars int
}

type Services struct {
	Core      *CoreService
	Settings  *SettingsService
	Profiles  *ProfileService
	Privilege *PrivilegeService
	Routing   *RoutingService
}

func NewTray(app *application.App, window application.Window, deps Services) *Tray {
	glyphs := trayGlyphs{off: trayicon.Disconnected(), logo: trayicon.Logo()}
	for lit := range glyphs.on {
		glyphs.on[lit] = trayicon.Connected(lit)
	}
	t := &Tray{
		app:       app,
		window:    window,
		tray:      app.SystemTray.New(),
		menu:      app.NewMenu(),
		glyphs:    glyphs,
		core:      deps.Core,
		settings:  deps.Settings,
		profiles:  deps.Profiles,
		privilege: deps.Privilege,
		routing:   deps.Routing,
		dirty:     make(chan struct{}, 1),
	}
	t.tray.SetMenu(t.menu)
	t.refresh()

	for _, name := range []string{EventStatus, EventProfiles, EventSettings, EventRouting, EventLatency} {
		app.Event.On(name, func(*application.CustomEvent) { t.invalidate() })
	}
	app.Event.OnApplicationEvent(events.Common.ApplicationStarted, func(*application.ApplicationEvent) {
		go t.loop()
		t.invalidate()
	})
	return t
}

func (t *Tray) invalidate() {
	select {
	case t.dirty <- struct{}{}:
	default:
	}
}

func (t *Tray) loop() {
	for range t.dirty {
		t.refresh()
	}
}

type traySnapshot struct {
	text     trayText
	settings settings.Settings
	state    StatusEvent
	profiles []ProfileView
	routes   []RoutingProfileView
	activeID string
	primary  *GroupView
	server   string
	delay    *int
}

func (t *Tray) snapshot() traySnapshot {
	st := traySnapshot{
		settings: t.settings.Get(),
		state:    t.core.State(),
		profiles: t.profiles.List(),
		routes:   t.routing.Profiles(),
	}
	st.text = trayTextFor(st.settings.Locale)
	for _, p := range st.profiles {
		if p.Active {
			st.activeID = p.ID
		}
	}
	if st.activeID == "" {
		return st
	}
	groups, err := t.profiles.Groups(st.activeID)
	if err != nil {
		return st
	}
	var server NodeView
	st.primary, server = activeServer(groups)
	st.server, st.delay = server.Name, server.Delay
	return st
}

func (st traySnapshot) icon() trayIcon {
	if st.settings.TrayIcon == settings.TrayIconLogo {
		return trayIcon{logo: true}
	}
	if st.state.Status != core.StatusRunning {
		return trayIcon{bars: -1}
	}
	if st.settings.TrayIcon == settings.TrayIconFlag && runtime.GOOS == "darwin" && st.server != "" {
		return trayIcon{flag: cmp.Or(flagOf(st.server), genericFlag)}
	}
	return trayIcon{bars: signalBars(st.delay)}
}

func signalBars(delay *int) int {
	switch {
	case delay == nil:
		return trayicon.Bars
	case *delay == 0:
		return 0
	case *delay < 150:
		return trayicon.Bars
	case *delay < 400:
		return trayicon.Bars - 1
	default:
		return 1
	}
}

const genericFlag = "🌐"

func flagOf(name string) string {
	runes := []rune(name)
	for i := 0; i+1 < len(runes); i++ {
		if unicode.Is(unicode.Regional_Indicator, runes[i]) && unicode.Is(unicode.Regional_Indicator, runes[i+1]) {
			return string(runes[i : i+2])
		}
	}
	return ""
}

func (t *Tray) refresh() {
	t.mu.Lock()
	defer t.mu.Unlock()
	st := t.snapshot()
	t.show(st.icon())
	t.tray.SetTooltip("ctun · " + st.statusLine())
	t.menu.Clear()
	t.build(st)
	t.menu.Update()
}

func (st traySnapshot) statusLine() string {
	tx := st.text
	switch st.state.Status {
	case core.StatusRunning:
		if st.server != "" {
			return tx.connected + " · " + st.server
		}
		return tx.connected
	case core.StatusStarting:
		return tx.connecting
	case core.StatusStopping:
		return tx.disconnecting
	}
	if st.state.Error != "" {
		return tx.failed
	}
	return tx.disconnected
}

func (t *Tray) build(st traySnapshot) {
	tx, m := st.text, t.menu
	m.Add(st.statusLine()).SetEnabled(false)
	switch st.state.Status {
	case core.StatusRunning:
		m.Add(tx.disconnect).OnClick(t.do(t.core.Stop))
	case core.StatusStopped:
		m.Add(tx.connect).SetEnabled(st.activeID != "").OnClick(t.do(t.connect))
	}
	m.AddSeparator()

	servers := m.AddSubmenu(tx.server)
	if st.primary == nil {
		servers.Add(tx.empty).SetEnabled(false)
	} else {
		group := st.primary.Name
		for _, node := range st.primary.Nodes {
			label := node.Name
			if label == config.AutoGroup {
				label = tx.auto
			}
			name := node.Name
			servers.AddRadio(label, node.Name == st.primary.Now).OnClick(t.do(func() error {
				return t.profiles.Select(st.activeID, group, name)
			}))
		}
	}

	subscriptions := m.AddSubmenu(tx.subscription)
	for _, p := range st.profiles {
		id := p.ID
		subscriptions.AddRadio(p.Name, p.Active).OnClick(t.do(func() error { return t.profiles.Activate(id) }))
	}
	if len(st.profiles) == 0 {
		subscriptions.Add(tx.empty).SetEnabled(false)
	}

	modes := m.AddSubmenu(tx.mode)
	for _, mode := range []struct {
		value settings.TunnelMode
		label string
	}{{settings.TunnelTUN, "TUN"}, {settings.TunnelProxy, tx.proxy}} {
		modes.AddRadio(mode.label, st.settings.TunnelMode == mode.value).
			OnClick(t.do(func() error { return t.setTunnelMode(mode.value) }))
	}

	routing := m.AddSubmenu(tx.routing)
	for _, mode := range []struct {
		value settings.RoutingMode
		label string
	}{{settings.RoutingRule, tx.rule}, {settings.RoutingGlobal, tx.global}, {settings.RoutingDirect, tx.direct}} {
		routing.AddRadio(mode.label, st.settings.RoutingMode == mode.value).OnClick(t.do(func() error {
			_, err := t.settings.Update(SettingsPatch{RoutingMode: &mode.value})
			return err
		}))
	}
	routing.AddSeparator()
	for _, r := range st.routes {
		id := r.Profile.ID
		routing.AddRadio(r.Profile.Name, r.Active).OnClick(t.do(func() error { return t.routing.ActivateProfile(id) }))
	}

	m.AddSeparator()
	m.Add(tx.show).OnClick(func(*application.Context) { t.window.Show().Focus() })
	m.Add(tx.quit).SetAccelerator("CmdOrCtrl+Q").OnClick(func(*application.Context) { t.app.Quit() })
}

func (t *Tray) connect() error {
	return t.withHelper(t.core.Start())
}

func (t *Tray) setTunnelMode(mode settings.TunnelMode) error {
	_, err := t.settings.Update(SettingsPatch{TunnelMode: &mode})
	return t.withHelper(err)
}

func (t *Tray) withHelper(err error) error {
	if !errors.Is(err, errHelperMissing) && !errors.Is(err, errHelperOutdated) {
		return err
	}
	if err := t.privilege.Install(); err != nil {
		return err
	}
	return t.core.Start()
}

func (t *Tray) do(action func() error) func(*application.Context) {
	return func(*application.Context) {
		go func() {
			if err := action(); err != nil {
				slog.Warn("tray action failed", "err", err)
			}
		}()
	}
}

func (t *Tray) show(icon trayIcon) {
	if t.hasShown && t.shown == icon {
		return
	}
	t.shown, t.hasShown = icon, true
	glyph := t.glyphs.off
	switch {
	case icon.logo:
		glyph = t.glyphs.logo
	case icon.bars >= 0:
		glyph = t.glyphs.on[icon.bars]
	}
	if runtime.GOOS != "darwin" {
		t.tray.SetIcon(glyph.Black).SetDarkModeIcon(glyph.White)
		return
	}
	// The image stays set but hidden behind the flag, as the status item cannot drop it.
	position := application.IconPosition(application.NSImageLeading)
	if icon.flag != "" {
		position = application.NSImageNone
	}
	t.tray.SetIconPosition(position)
	t.tray.SetTemplateIcon(glyph.Black)
	t.tray.SetLabel(icon.flag)
}
