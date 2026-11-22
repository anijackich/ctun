package config

import (
	"errors"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
)

// DefaultNameservers are DoH by IP, so no bootstrap resolver is needed. The core must resolve
// on its own: the system resolver may be another VPN's fake-ip DNS.
var DefaultNameservers = []string{"https://1.1.1.1/dns-query", "https://8.8.8.8/dns-query"}

var ErrNameserver = errors.New("config: a nameserver is an IP address, optionally with udp://, tcp://, tls://, https:// or quic://")

var nameserverPorts = map[string]uint16{"udp": 53, "tcp": 53, "tls": 853, "https": 443, "quic": 853}

// Nameserver is a DNS server entry as mihomo writes it, limited to IP hosts: a hostname would
// need a bootstrap resolver, which mihomo otherwise fills with public defaults of its own.
type Nameserver struct {
	Scheme string
	Addr   netip.AddrPort

	Path string
}

func ParseNameserver(raw string) (Nameserver, error) {
	raw = strings.TrimSpace(raw)
	if ip, err := netip.ParseAddr(raw); err == nil {
		return Nameserver{Scheme: "udp", Addr: netip.AddrPortFrom(ip, nameserverPorts["udp"])}, nil
	}
	if !strings.Contains(raw, "://") {
		raw = "udp://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return Nameserver{}, ErrNameserver
	}
	port, ok := nameserverPorts[u.Scheme]
	if !ok || (u.Scheme != "https" && u.Path != "") {
		return Nameserver{}, ErrNameserver
	}
	ip, err := netip.ParseAddr(u.Hostname())
	if err != nil {
		return Nameserver{}, ErrNameserver
	}
	if p := u.Port(); p != "" {
		n, err := strconv.ParseUint(p, 10, 16)
		if err != nil || n == 0 {
			return Nameserver{}, ErrNameserver
		}
		port = uint16(n)
	}
	return Nameserver{Scheme: u.Scheme, Addr: netip.AddrPortFrom(ip.Unmap(), port), Path: u.Path}, nil
}

func (n Nameserver) String() string {
	defaultPort := n.Addr.Port() == nameserverPorts[n.Scheme]
	if n.Scheme == "udp" && defaultPort {
		return n.Addr.Addr().String()
	}
	host := n.Addr.String()
	if defaultPort {
		host = n.Addr.Addr().String()
		if n.Addr.Addr().Is6() {
			host = "[" + host + "]"
		}
	}
	return (&url.URL{Scheme: n.Scheme, Host: host, Path: n.Path}).String()
}
