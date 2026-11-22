//go:build subscription

package profile

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"ctun/internal/config"
	"ctun/internal/latency"
)

// TestRealSubscription checks a real provider end to end without printing the link:
//
//	CTUN_SUBSCRIPTION_FILE=subscription.txt go test -tags subscription -run Real -v ./internal/profile
func TestRealSubscription(t *testing.T) {
	path := os.Getenv("CTUN_SUBSCRIPTION_FILE")
	if path == "" {
		t.Skip("CTUN_SUBSCRIPTION_FILE is not set")
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join("..", "..", path)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	link := strings.TrimSpace(strings.SplitN(string(raw), "\n", 2)[0])

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	resp, err := (&Fetcher{}).Fetch(ctx, link, "mihomo/v1.19.34")
	if err != nil {
		t.Fatal(err)
	}
	meta := ParseHeaders(resp.Header)
	layer, err := Parse(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	kinds := map[string]int{}
	for _, p := range layer.Proxies {
		kinds[Describe(p)]++
	}
	t.Logf("title=%q interval=%dh expire=%d total=%d support=%t site=%t announce=%t",
		meta.Title, meta.UpdateIntervalHours, meta.Info.Expire, meta.Info.Total,
		meta.Info.SupportURL != "", meta.Info.WebPageURL != "", meta.Info.Announce != "")
	t.Logf("servers=%d kinds=%v groups=%d", len(layer.Proxies), kinds, len(config.Groups(layer)))

	cfg, err := config.Build(config.Options{
		RoutingMode: "rule",
		LogLevel:    "warning",
		Controller:  config.Controller{Addr: "127.0.0.1:1", Secret: "x"},
		Profile:     layer,
	})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, cfg, 0o600); err != nil {
		t.Fatal(err)
	}
	core := filepath.Join("..", "..", "core", "mihomo-"+goosArch())
	out, err := exec.CommandContext(ctx, core, "-t", "-d", dir, "-f", cfgPath).CombinedOutput()
	if err != nil {
		t.Fatalf("mihomo -t failed: %s", lastLine(out))
	}
	t.Logf("mihomo -t: %s", lastLine(out))

	names := make([]string, 0, len(layer.Proxies))
	for _, p := range layer.Proxies {
		names = append(names, p["name"].(string))
	}
	start := time.Now()
	var mu sync.Mutex
	reachable, first := 0, time.Duration(0)
	latency.Run(ctx, names, latency.Direct(layer.Proxies), func(_ string, delay int) {
		mu.Lock()
		defer mu.Unlock()
		if first == 0 {
			first = time.Since(start)
		}
		if delay > 0 {
			reachable++
		}
	})
	t.Logf("direct latency test: %d/%d reachable, first result after %s, all in %s",
		reachable, len(names), first.Round(time.Millisecond), time.Since(start).Round(time.Millisecond))
}

func goosArch() string {
	out, _ := exec.Command("go", "env", "GOOS", "GOARCH").Output()
	return strings.Join(strings.Fields(string(out)), "-")
}

func lastLine(b []byte) string {
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	line := lines[len(lines)-1]
	if i := strings.Index(line, "test is"); i >= 0 {
		return line[i:]
	}
	return line
}
