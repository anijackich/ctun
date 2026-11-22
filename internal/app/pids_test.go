package app

import (
	"context"
	"testing"
)

func TestParseLsof(t *testing.T) {
	out := "p100\nf5\nn127.0.0.1:50301->127.0.0.1:7890\nf6\nn*:5353\np200\nf9\nn[::1]:50400->[::1]:443\nn127.0.0.1:7890\n"
	owners := parseLsof(out)
	for port, want := range map[string]int{"50301": 100, "5353": 100, "50400": 200, "7890": 200} {
		if owners[port] != want {
			t.Errorf("port %s → %d, want %d", port, owners[port], want)
		}
	}
}

func TestPIDCacheScansOnlyForNewConnections(t *testing.T) {
	scans := 0
	c := &pidCache{byID: map[string]int{}, scan: func(context.Context) map[string]int {
		scans++
		return map[string]int{"1000": 42}
	}}
	if got := c.resolve(map[string]string{"a": "1000", "b": "2000"}); got["a"] != 42 || got["b"] != 0 {
		t.Fatalf("pids = %v", got)
	}
	c.resolve(map[string]string{"a": "1000", "b": "2000"})
	if scans != 1 {
		t.Fatalf("known connections rescanned: %d scans", scans)
	}
	c.resolve(map[string]string{"a": "1000"})
	if _, ok := c.byID["b"]; ok || scans != 1 {
		t.Fatalf("closed connection kept or rescanned: %v, %d scans", c.byID, scans)
	}
}
