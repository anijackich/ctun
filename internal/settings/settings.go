package settings

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"strings"
	"sync"

	"ctun/internal/config"
	"ctun/internal/fsutil"
)

type TunnelMode string

const (
	TunnelTUN   TunnelMode = "tun"
	TunnelProxy TunnelMode = "proxy"
)

type RoutingMode string

const (
	RoutingRule   RoutingMode = "rule"
	RoutingGlobal RoutingMode = "global"
	RoutingDirect RoutingMode = "direct"
)

var (
	logLevels = []string{"error", "warning", "info", "debug"}
	tunStacks = []string{"system", "gvisor", "mixed"}
	themes    = []string{"", "light", "dark"}
	trayIcons = []string{TrayIconSignal, TrayIconFlag, TrayIconLogo}
)

const (
	TrayIconSignal = ""

	TrayIconFlag = "flag"

	TrayIconLogo = "logo"
)

const (
	minPort           = 1024
	maxPort           = 65535
	minInspectorWidth = 280
	maxInspectorWidth = 520
)

type Settings struct {
	Locale      string      `json:"locale"`
	Theme       string      `json:"theme"`
	TrayIcon    string      `json:"trayIcon"`
	TunnelMode  TunnelMode  `json:"tunnelMode"`
	RoutingMode RoutingMode `json:"routingMode"`
	MixedPort   int         `json:"mixedPort"`
	PortAuth    bool        `json:"portAuth"`
	// SystemProxy points the OS proxy at the mixed port in proxy mode. Browsers cannot
	// authenticate to a system proxy, so it is mutually exclusive with PortAuth.
	SystemProxy    bool   `json:"systemProxy"`
	LogLevel       string `json:"logLevel"`
	TunStack       string `json:"tunStack"`
	InspectorOpen  bool   `json:"inspectorOpen"`
	InspectorWidth int    `json:"inspectorWidth"`
	DNS            DNS    `json:"dns"`
}

type DNS struct {
	Nameservers []string `json:"nameservers"`

	Direct []string `json:"direct"`

	Servers  []string `json:"servers"`
	ViaProxy bool     `json:"viaProxy"`
	// SystemOverride points the OS resolver into the tunnel while TUN runs: a resolver on the
	// LAN is routed around it, and domain rules then see no queries.
	SystemOverride bool `json:"systemOverride"`
}

func DefaultDNS() DNS {
	return DNS{
		Nameservers:    slices.Clone(config.DefaultNameservers),
		Direct:         slices.Clone(config.DefaultNameservers),
		Servers:        slices.Clone(config.DefaultNameservers),
		ViaProxy:       true,
		SystemOverride: true,
	}
}

func (d DNS) SameResolvers(other DNS) bool {
	return slices.Equal(d.Nameservers, other.Nameservers) && slices.Equal(d.Direct, other.Direct) &&
		slices.Equal(d.Servers, other.Servers) && d.ViaProxy == other.ViaProxy
}

func (d DNS) SystemServer() string {
	ns, err := config.ParseNameserver(d.Direct[0])
	if err != nil {
		return ""
	}
	return ns.Addr.Addr().String()
}

func (d DNS) normalize() DNS {
	clean := func(list []string) []string {
		var out []string
		for _, raw := range list {
			if ns, err := config.ParseNameserver(raw); err == nil && !slices.Contains(out, ns.String()) {
				out = append(out, ns.String())
			}
		}
		if len(out) == 0 {
			return slices.Clone(config.DefaultNameservers)
		}
		return out
	}
	d.Nameservers, d.Direct, d.Servers = clean(d.Nameservers), clean(d.Direct), clean(d.Servers)
	return d
}

func InvalidNameservers(lines []string) []int {
	var invalid []int
	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if _, err := config.ParseNameserver(line); err != nil {
			invalid = append(invalid, i+1)
		}
	}
	return invalid
}

func Default() Settings {
	return Settings{
		TunnelMode:     TunnelTUN,
		RoutingMode:    RoutingRule,
		MixedPort:      7890,
		PortAuth:       true,
		LogLevel:       "warning",
		TunStack:       "mixed",
		InspectorOpen:  true,
		InspectorWidth: 320,
		DNS:            DefaultDNS(),
	}
}

func (s Settings) normalize() Settings {
	def := Default()
	if s.TunnelMode != TunnelTUN && s.TunnelMode != TunnelProxy {
		s.TunnelMode = def.TunnelMode
	}
	if s.RoutingMode != RoutingRule && s.RoutingMode != RoutingGlobal && s.RoutingMode != RoutingDirect {
		s.RoutingMode = def.RoutingMode
	}
	if s.MixedPort < minPort || s.MixedPort > maxPort {
		s.MixedPort = def.MixedPort
	}
	if !slices.Contains(logLevels, s.LogLevel) {
		s.LogLevel = def.LogLevel
	}
	if !slices.Contains(tunStacks, s.TunStack) {
		s.TunStack = def.TunStack
	}
	if !slices.Contains(themes, s.Theme) {
		s.Theme = ""
	}
	if !slices.Contains(trayIcons, s.TrayIcon) {
		s.TrayIcon = TrayIconSignal
	}
	s.InspectorWidth = min(maxInspectorWidth, max(minInspectorWidth, s.InspectorWidth))
	s.DNS = s.DNS.normalize()
	if s.PortAuth {
		s.SystemProxy = false
	}
	return s
}

type Store struct {
	path    string
	mu      sync.Mutex
	current Settings
}

func Open(path string) (*Store, error) {
	current := Default()
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, fmt.Errorf("read settings: %w", err)
	default:
		if err := json.Unmarshal(data, &current); err != nil {
			return nil, fmt.Errorf("parse settings: %w", err)
		}
	}
	return &Store{path: path, current: current.normalize()}, nil
}

func (s *Store) Get() Settings {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.current
}

func (s *Store) Update(fn func(*Settings)) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := s.current
	fn(&next)
	next = next.normalize()

	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return s.current, fmt.Errorf("encode settings: %w", err)
	}
	if err := fsutil.WriteFileAtomic(s.path, data, 0o600); err != nil {
		return s.current, fmt.Errorf("save settings: %w", err)
	}
	s.current = next
	return next, nil
}
