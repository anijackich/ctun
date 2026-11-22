package config

import "testing"

func TestParseNameserver(t *testing.T) {
	valid := map[string]string{
		"1.1.1.1":                      "1.1.1.1",
		"8.8.8.8:5353":                 "udp://8.8.8.8:5353",
		"tcp://9.9.9.9:53":             "tcp://9.9.9.9",
		"tls://1.1.1.1":                "tls://1.1.1.1",
		"https://1.1.1.1/dns-query":    "https://1.1.1.1/dns-query",
		"https://1.1.1.1:8443/dns":     "https://1.1.1.1:8443/dns",
		"quic://[2606:4700::1111]:784": "quic://[2606:4700::1111]:784",
		"udp://[2001:4860:4860::8888]": "2001:4860:4860::8888",
	}
	if ns, err := ParseNameserver("\ttls://9.9.9.9:853 "); err != nil || ns.String() != "tls://9.9.9.9" {
		t.Errorf("surrounding space: %v, %v", ns, err)
	}
	for raw, want := range valid {
		ns, err := ParseNameserver(raw)
		if err != nil {
			t.Errorf("%q: %v", raw, err)
			continue
		}
		if got := ns.String(); got != want {
			t.Errorf("%q → %q, want %q", raw, got, want)
		}
	}
	for _, raw := range []string{
		"", "dns.google", "https://dns.google/dns-query", "dhcp://en0", "system",
		"tls://1.1.1.1/path", "https://1.1.1.1/dns-query#PROXY", "udp://1.1.1.1:0", "ftp://1.1.1.1",
	} {
		if _, err := ParseNameserver(raw); err == nil {
			t.Errorf("%q must be rejected", raw)
		}
	}
}
