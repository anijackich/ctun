package core

import "testing"

func TestParseVersion(t *testing.T) {
	cases := map[string]string{
		"Mihomo Meta v1.19.34 darwin arm64 with go1.26.11 Wed Nov 11\nUse tags: with_gvisor\n": "v1.19.34",
		"Mihomo Meta alpha-abc1234 linux amd64":                                                "",
		"":                                                                                     "",
	}
	for out, want := range cases {
		if got := parseVersion(out); got != want {
			t.Errorf("parseVersion(%q) = %q, want %q", out, got, want)
		}
	}
}
