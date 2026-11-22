package app

import (
	"testing"

	"ctun/internal/core"
	"ctun/internal/settings"
)

func TestSystemProxyAndPortAuthExcludeEachOther(t *testing.T) {
	s := newTestService(t, proxyMode, false)
	ss := NewSettingsService(s.settings, s)
	on, off := true, false

	got, err := ss.Update(SettingsPatch{SystemProxy: &on})
	if err != nil {
		t.Fatal(err)
	}
	if !got.SystemProxy || got.PortAuth {
		t.Fatalf("system proxy must drop the password: %+v", got)
	}
	if got, err = ss.Update(SettingsPatch{PortAuth: &on}); err != nil {
		t.Fatal(err)
	}
	if got.SystemProxy || !got.PortAuth {
		t.Fatalf("password must switch the system proxy off: %+v", got)
	}
	if got, err = ss.Update(SettingsPatch{SystemProxy: &off}); err != nil || !got.PortAuth {
		t.Fatalf("turning the system proxy off keeps the password: %+v %v", got, err)
	}
}

func TestSystemProxyTarget(t *testing.T) {
	s := newTestService(t, func(st *settings.Settings) {
		st.TunnelMode, st.PortAuth, st.SystemProxy = settings.TunnelProxy, false, true
	}, false)
	if got := s.systemProxyTarget(); got != (proxyTarget{}) {
		t.Fatalf("stopped core → %+v, want off", got)
	}
	s.client, s.creds = new(core.Client), runtimeCreds{port: 7890}
	if got := s.systemProxyTarget(); got != (proxyTarget{port: 7890, bypassLAN: true}) {
		t.Fatalf("running core → %+v", got)
	}
	p := s.routes.Active()
	p.ProxyLAN = true
	if _, err := s.routes.Save(p); err != nil {
		t.Fatal(err)
	}
	if got := s.systemProxyTarget(); got != (proxyTarget{port: 7890}) {
		t.Fatalf("proxied local network → %+v", got)
	}
	s.creds.password = "x"
	if got := s.systemProxyTarget(); got.port != 0 {
		t.Fatalf("password-protected port → %+v, want off", got)
	}
	s.creds.password = ""
	if _, err := s.settings.Update(func(st *settings.Settings) { st.TunnelMode = settings.TunnelTUN }); err != nil {
		t.Fatal(err)
	}
	if got := s.systemProxyTarget(); got.port != 0 {
		t.Fatalf("TUN mode → %+v, want off", got)
	}
}
