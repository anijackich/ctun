package app

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log/slog"
	"net"
	"net/url"
	"strconv"

	"ctun/internal/config"
	"ctun/internal/core"
	"ctun/internal/routing"
	"ctun/internal/settings"
)

const proxyUser = "ctun"

type runtimeCreds struct {
	controllerAddr string
	secret         string
	port           int
	password       string
}

func newRuntimeCreds(opts settings.Settings) (runtimeCreds, error) {
	addr, err := freeLocalAddr()
	if err != nil {
		return runtimeCreds{}, err
	}
	secret, err := randomToken()
	if err != nil {
		return runtimeCreds{}, err
	}
	rt := runtimeCreds{controllerAddr: addr, secret: secret}
	if opts.TunnelMode == settings.TunnelTUN {
		return rt, nil
	}
	rt.port = opts.MixedPort
	if opts.PortAuth {
		if rt.password, err = randomToken(); err != nil {
			return runtimeCreds{}, err
		}
	}
	return rt, nil
}

func (r runtimeCreds) proxyURL() string {
	if r.port == 0 {
		return ""
	}
	u := url.URL{Scheme: "http", Host: net.JoinHostPort("127.0.0.1", strconv.Itoa(r.port))}
	if r.password != "" {
		u.User = url.UserPassword(proxyUser, r.password)
	}
	return u.String()
}

func (s *CoreService) buildConfig(opts settings.Settings, creds runtimeCreds) ([]byte, map[string]string, error) {
	id := s.profiles.ActiveID()
	if id == "" {
		return nil, nil, s.fail(CodeNoProfile, errNoProfile)
	}
	p, err := s.profiles.Get(id)
	if err != nil {
		return nil, nil, s.fail(CodeNoProfile, err)
	}
	layer, err := s.profiles.Layer(id)
	if err != nil {
		return nil, nil, s.fail(CodeProfileInvalid, err)
	}

	cfgOpts := config.Options{
		RoutingMode:    string(opts.RoutingMode),
		LogLevel:       opts.LogLevel,
		MixedPort:      creds.port,
		AllowAnonymous: creds.password == "" && creds.port != 0,
		Controller:     config.Controller{Addr: creds.controllerAddr, Secret: creds.secret},
		Profile:        layer,
		Routing:        userRouting(s.routes.Active(), layer),
		DNS: config.DNS{
			Nameservers: opts.DNS.Nameservers,
			Direct:      opts.DNS.Direct,
			Servers:     opts.DNS.Servers,
			ViaProxy:    opts.DNS.ViaProxy,
		},
	}
	if creds.password != "" {
		cfgOpts.Auth = &config.Credentials{User: proxyUser, Password: creds.password}
	}
	if opts.TunnelMode == settings.TunnelTUN {
		cfgOpts.TUN = &config.TUN{Stack: opts.TunStack}
	}
	cfg, err := config.Build(cfgOpts)
	if err != nil {
		return nil, nil, s.fail(CodeCoreFailed, err)
	}
	slog.Info("core config built", "mode", opts.TunnelMode, "routing", opts.RoutingMode, "subscription", p.Name)
	return cfg, p.Selected, nil
}

func userRouting(p routing.Profile, layer *config.Profile) *config.Routing {
	c := routing.Compile(p, config.MainGroup(layer))
	return &config.Routing{
		Rules:     c.Rules,
		Providers: c.Providers,
		Final:     c.Final,
		ProxyLAN:  c.ProxyLAN,
	}
}

func applySelections(client *core.Client, selected map[string]string) {
	for group, name := range selected {
		ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
		if err := client.SelectProxy(ctx, group, name); err != nil {
			slog.Debug("restore selection", "group", group, "err", err)
		}
		cancel()
	}
}

func freeLocalAddr() (string, error) {
	var lc net.ListenConfig
	l, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("reserve controller port: %w", err)
	}
	defer l.Close()
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(l.Addr().(*net.TCPAddr).Port)), nil
}

func randomToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("generate secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
