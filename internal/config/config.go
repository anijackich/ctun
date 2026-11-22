package config

import (
	"errors"
	"fmt"

	"go.yaml.in/yaml/v3"
)

type Controller struct {
	Addr   string
	Secret string
}

type Credentials struct {
	User     string
	Password string
}

type Options struct {
	RoutingMode string
	LogLevel    string

	MixedPort int

	Auth           *Credentials
	AllowAnonymous bool
	Controller     Controller

	Profile *Profile

	Routing *Routing

	TUN *TUN
	DNS DNS
}

type DNS struct {
	Nameservers []string

	Direct []string

	Servers  []string
	ViaProxy bool
}

type TUN struct {
	Stack string
}

var ErrPortWithoutAuth = errors.New("config: mixed port requires credentials or AllowAnonymous")

type document struct {
	Mode               string           `yaml:"mode"`
	LogLevel           string           `yaml:"log-level"`
	FindProcessMode    string           `yaml:"find-process-mode"`
	IPv6               bool             `yaml:"ipv6"`
	AllowLAN           bool             `yaml:"allow-lan"`
	BindAddress        string           `yaml:"bind-address,omitempty"`
	MixedPort          int              `yaml:"mixed-port,omitempty"`
	Authentication     []string         `yaml:"authentication,omitempty"`
	ExternalController string           `yaml:"external-controller"`
	Secret             string           `yaml:"secret"`
	Profile            profile          `yaml:"profile"`
	TUN                *tun             `yaml:"tun,omitempty"`
	Sniffer            *sniffer         `yaml:"sniffer,omitempty"`
	DNS                dns              `yaml:"dns"`
	Proxies            []map[string]any `yaml:"proxies"`
	ProxyGroups        []map[string]any `yaml:"proxy-groups"`
	ProxyProviders     map[string]any   `yaml:"proxy-providers,omitempty"`
	RuleProviders      map[string]any   `yaml:"rule-providers,omitempty"`
	Rules              []string         `yaml:"rules"`
}

type profile struct {
	StoreSelected bool `yaml:"store-selected"`
	StoreFakeIP   bool `yaml:"store-fake-ip"`
}

type dns struct {
	Enable                bool     `yaml:"enable"`
	IPv6                  bool     `yaml:"ipv6"`
	EnhancedMode          string   `yaml:"enhanced-mode,omitempty"`
	FakeIPFilter          []string `yaml:"fake-ip-filter,omitempty"`
	RespectRules          bool     `yaml:"respect-rules"`
	DefaultNameserver     []string `yaml:"default-nameserver"`
	Nameserver            []string `yaml:"nameserver"`
	DirectNameserver      []string `yaml:"direct-nameserver"`
	ProxyServerNameserver []string `yaml:"proxy-server-nameserver"`
}

type sniffer struct {
	Enable          bool                     `yaml:"enable"`
	OverrideDest    bool                     `yaml:"override-destination"`
	ForceDNSMapping bool                     `yaml:"force-dns-mapping"`
	ParsePureIP     bool                     `yaml:"parse-pure-ip"`
	SkipDomain      []string                 `yaml:"skip-domain"`
	Sniff           map[string]sniffProtocol `yaml:"sniff"`
}

type sniffProtocol struct {
	Ports []string `yaml:"ports"`
}

type tun struct {
	Enable              bool     `yaml:"enable"`
	Stack               string   `yaml:"stack"`
	AutoRoute           bool     `yaml:"auto-route"`
	AutoDetectInterface bool     `yaml:"auto-detect-interface"`
	DNSHijack           []string `yaml:"dns-hijack"`
}

func Build(opts Options) ([]byte, error) {
	doc := document{
		Mode:     opts.RoutingMode,
		LogLevel: opts.LogLevel,

		FindProcessMode:    "always",
		ExternalController: opts.Controller.Addr,
		Secret:             opts.Controller.Secret,
		DNS:                buildDNS(opts.DNS),
		Proxies:            []map[string]any{},
		ProxyGroups:        []map[string]any{{"name": PrimaryGroup, "type": "select", "proxies": []string{"DIRECT"}}},
		Rules:              []string{"MATCH," + PrimaryGroup},
	}
	if p := opts.Profile; p != nil {
		doc.Proxies = p.Proxies
		doc.ProxyProviders = p.ProxyProviders
		doc.ProxyGroups, _, _ = ownRouting(p)
		doc.Rules, doc.RuleProviders = Rules(p, opts.Routing)
	}
	if opts.TUN != nil {
		applyTUN(&doc, opts.TUN)
	}

	if opts.MixedPort != 0 {
		doc.MixedPort = opts.MixedPort
		doc.BindAddress = "127.0.0.1"
		switch {
		case opts.Auth != nil:
			doc.Authentication = []string{opts.Auth.User + ":" + opts.Auth.Password}
		case !opts.AllowAnonymous:
			return nil, ErrPortWithoutAuth
		}
	}

	out, err := yaml.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("config: encode: %w", err)
	}
	return out, nil
}

// buildDNS sets every resolver mihomo would otherwise fill with public servers of its own
// choosing, including default-nameserver: unused with IP-only entries, but still required.
func buildDNS(opts DNS) dns {
	direct := orDefault(opts.Direct)
	return dns{
		Enable: true,

		RespectRules:          opts.ViaProxy,
		DefaultNameserver:     direct,
		Nameserver:            orDefault(opts.Nameservers),
		DirectNameserver:      direct,
		ProxyServerNameserver: orDefault(opts.Servers),
	}
}

func orDefault(list []string) []string {
	if len(list) == 0 {
		return DefaultNameservers
	}
	return list
}

func applyTUN(doc *document, opts *TUN) {
	doc.TUN = &tun{
		Enable:              true,
		Stack:               opts.Stack,
		AutoRoute:           true,
		AutoDetectInterface: true,
		DNSHijack:           []string{"any:53"},
	}
	doc.DNS.EnhancedMode = "fake-ip"
	doc.DNS.FakeIPFilter = []string{"*.lan", "+.local", "+.localhost"}
	// Domains come from the traffic itself, so rules match and proxied connections go to the
	// domain even when an app's DNS bypassed the tunnel (own DoH, a resolver on the LAN).
	doc.Sniffer = &sniffer{
		Enable:          true,
		OverrideDest:    true,
		ForceDNSMapping: true,
		ParsePureIP:     true,
		// APNs presents certificates for odd names; rewriting its destination breaks pushes.
		SkipDomain: []string{"+.push.apple.com"},
		Sniff: map[string]sniffProtocol{
			"HTTP": {Ports: []string{"80", "8080-8880"}},
			"TLS":  {Ports: []string{"443", "8443"}},
			"QUIC": {Ports: []string{"443", "8443"}},
		},
	}
}
