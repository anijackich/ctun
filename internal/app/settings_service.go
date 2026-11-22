package app

import "ctun/internal/settings"

type SettingsPatch struct {
	Locale         *string               `json:"locale,omitempty"`
	Theme          *string               `json:"theme,omitempty"`
	TrayIcon       *string               `json:"trayIcon,omitempty"`
	TunnelMode     *settings.TunnelMode  `json:"tunnelMode,omitempty"`
	RoutingMode    *settings.RoutingMode `json:"routingMode,omitempty"`
	MixedPort      *int                  `json:"mixedPort,omitempty"`
	PortAuth       *bool                 `json:"portAuth,omitempty"`
	SystemProxy    *bool                 `json:"systemProxy,omitempty"`
	LogLevel       *string               `json:"logLevel,omitempty"`
	TunStack       *string               `json:"tunStack,omitempty"`
	InspectorOpen  *bool                 `json:"inspectorOpen,omitempty"`
	InspectorWidth *int                  `json:"inspectorWidth,omitempty"`
	DNS            *settings.DNS         `json:"dns,omitempty"`
}

type SettingsService struct {
	store *settings.Store
	core  *CoreService
}

func NewSettingsService(store *settings.Store, core *CoreService) *SettingsService {
	return &SettingsService{store: store, core: core}
}

func (s *SettingsService) Get() settings.Settings {
	return s.store.Get()
}

func (s *SettingsService) Update(patch SettingsPatch) (settings.Settings, error) {
	before := s.store.Get()
	after, err := s.store.Update(func(st *settings.Settings) { patch.apply(st) })
	if err != nil {
		return before, err
	}
	emit(EventSettings, after)
	s.core.syncSystemSettings()

	if after.RoutingMode != before.RoutingMode {
		if err := s.core.applyRoutingMode(after.RoutingMode); err != nil {
			return after, err
		}
	}
	if after.TunnelMode != before.TunnelMode || after.MixedPort != before.MixedPort ||
		after.PortAuth != before.PortAuth || after.LogLevel != before.LogLevel ||
		after.TunStack != before.TunStack {
		return after, s.core.restart()
	}
	if !after.DNS.SameResolvers(before.DNS) {
		return after, s.core.reload()
	}
	return after, nil
}

func (s *SettingsService) ResetDNS() (settings.Settings, error) {
	dns := settings.DefaultDNS()
	return s.Update(SettingsPatch{DNS: &dns})
}

func (s *SettingsService) InvalidNameservers(lines []string) []int {
	return settings.InvalidNameservers(lines)
}

func (p SettingsPatch) apply(st *settings.Settings) {
	set(&st.Locale, p.Locale)
	set(&st.Theme, p.Theme)
	set(&st.TrayIcon, p.TrayIcon)
	set(&st.TunnelMode, p.TunnelMode)
	set(&st.RoutingMode, p.RoutingMode)
	set(&st.MixedPort, p.MixedPort)
	set(&st.PortAuth, p.PortAuth)
	set(&st.SystemProxy, p.SystemProxy)

	if p.SystemProxy != nil && *p.SystemProxy {
		st.PortAuth = false
	}
	if p.PortAuth != nil && *p.PortAuth {
		st.SystemProxy = false
	}
	set(&st.LogLevel, p.LogLevel)
	set(&st.TunStack, p.TunStack)
	set(&st.InspectorOpen, p.InspectorOpen)
	set(&st.InspectorWidth, p.InspectorWidth)
	set(&st.DNS, p.DNS)
}

func set[T any](dst *T, src *T) {
	if src != nil {
		*dst = *src
	}
}
