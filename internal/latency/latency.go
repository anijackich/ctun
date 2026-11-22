package latency

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/metacubex/mihomo/adapter"
	"github.com/metacubex/mihomo/component/dialer"
	"github.com/metacubex/mihomo/component/resolver"
	C "github.com/metacubex/mihomo/constant"
	"github.com/metacubex/mihomo/dns"

	"ctun/internal/config"
)

const (
	Timeout     = 5 * time.Second
	Concurrency = 16
)

type Tester func(ctx context.Context, name string) (int, error)

func Run(ctx context.Context, names []string, test Tester, report func(name string, delay int)) {
	sem := make(chan struct{}, Concurrency)
	var wg sync.WaitGroup
	for _, name := range names {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			return
		}
		wg.Add(1)
		go func() {
			defer func() { <-sem; wg.Done() }()
			testCtx, cancel := context.WithTimeout(ctx, Timeout)
			defer cancel()
			delay, err := test(testCtx, name)
			if err != nil {
				delay = 0
			}
			report(name, delay)
		}()
	}
	wg.Wait()
}

var (
	dnsMu        sync.Mutex
	installedDNS string
)

func useServerDNS(nameservers []string) {
	dnsMu.Lock()
	defer dnsMu.Unlock()
	key := strings.Join(nameservers, "\n")
	if key == installedDNS {
		return
	}
	installedDNS = key
	servers := make([]dns.NameServer, 0, len(nameservers))
	for _, raw := range nameservers {
		if ns, err := config.ParseNameserver(raw); err == nil {
			servers = append(servers, mihomoNameserver(ns))
		}
	}
	r := dns.NewResolver(dns.Config{Main: servers})
	resolver.DefaultResolver = r
	resolver.ProxyServerHostResolver = r.Resolver
	resolver.DirectHostResolver = r.Resolver
}

func mihomoNameserver(ns config.Nameserver) dns.NameServer {
	addr := ns.Addr.String()
	switch ns.Scheme {
	case "udp":
		return dns.NameServer{Addr: addr}
	case "https":
		return dns.NameServer{Net: "https", Addr: (&url.URL{Scheme: "https", Host: addr, Path: ns.Path}).String()}
	default:
		return dns.NameServer{Net: ns.Scheme, Addr: addr}
	}
}

// bindPhysical pins in-process dials to the physical uplink. With TUN on, the app's own
// traffic would otherwise enter our tunnel and measure servers through the current one.
func bindPhysical() {
	dialer.DefaultInterface.Store(physicalInterface())
}

func physicalInterface() string {
	ifaces, err := net.Interfaces()
	if err != nil {
		return ""
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&(net.FlagLoopback|net.FlagPointToPoint) != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil && !ipnet.IP.IsLinkLocalUnicast() {
				return iface.Name
			}
		}
	}
	return ""
}

func Direct(proxies []map[string]any, nameservers []string) Tester {
	useServerDNS(nameservers)
	bindPhysical()
	outbounds := make(map[string]C.Proxy, len(proxies))
	for _, mapping := range proxies {
		name, _ := mapping["name"].(string)
		if p, err := adapter.ParseProxy(mapping); err == nil && name != "" {
			outbounds[name] = p
		}
	}
	return func(ctx context.Context, name string) (int, error) {
		p, ok := outbounds[name]
		if !ok {
			return 0, fmt.Errorf("latency: no proxy %q", name)
		}
		delay, err := p.URLTest(ctx, config.TestURL, nil)
		if err != nil {
			return 0, fmt.Errorf("latency: %s: %w", name, err)
		}
		return int(delay), nil
	}
}
