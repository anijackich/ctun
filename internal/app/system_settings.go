package app

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"ctun/internal/settings"
	"ctun/internal/sysnet"
)

type proxyTarget struct {
	port      int
	bypassLAN bool
}

func (s *CoreService) SystemProxySupported() bool {
	return s.proxyManager.Supported()
}

func (s *CoreService) syncSystemSettings() {
	s.proxySync.trigger()
	s.dnsSync.trigger()
}

func (s *CoreService) tunRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client != nil && s.active == s.tun
}

func (s *CoreService) systemProxyTarget() proxyTarget {
	opts := s.settings.Get()
	if !opts.SystemProxy || opts.TunnelMode != settings.TunnelProxy {
		return proxyTarget{}
	}
	s.mu.Lock()
	port := s.creds.port
	if s.client == nil || s.creds.password != "" {
		port = 0
	}
	s.mu.Unlock()
	if port == 0 {
		return proxyTarget{}
	}
	return proxyTarget{port: port, bypassLAN: !s.routes.Active().ProxyLAN}
}

func applySystemProxy(m *sysnet.Manager[sysnet.Proxy]) func(context.Context, proxyTarget) error {
	return func(ctx context.Context, target proxyTarget) error {
		if target.port == 0 {
			if restored, err := m.Disable(ctx); err != nil || !restored {
				return err
			}
			slog.Info("system proxy restored")
			return nil
		}
		bypass := sysnet.LocalBypass
		if target.bypassLAN {
			bypass = append(slices.Clone(bypass), sysnet.LANBypass...)
		}
		if err := m.Enable(ctx, sysnet.Proxy{Host: "127.0.0.1", Port: target.port, Bypass: bypass}); err != nil {
			return err
		}
		slog.Info("system proxy set", "address", fmt.Sprintf("127.0.0.1:%d", target.port), "lan direct", target.bypassLAN)
		return nil
	}
}

func (s *CoreService) systemDNSTarget() string {
	dns := s.settings.Get().DNS
	if !dns.SystemOverride || !s.tunRunning() {
		return ""
	}
	return dns.SystemServer()
}

func applySystemDNS(m *sysnet.Manager[[]string]) func(context.Context, string) error {
	return func(ctx context.Context, server string) error {
		if server == "" {
			if restored, err := m.Disable(ctx); err != nil || !restored {
				return err
			}
			slog.Info("system DNS restored")
			return nil
		}
		if err := m.Enable(ctx, []string{server}); err != nil {
			return err
		}
		slog.Info("system DNS routed into the tunnel", "server", server)
		return nil
	}
}
