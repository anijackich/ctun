//go:build integration

package core_test

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"ctun/internal/config"
	"ctun/internal/core"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func startCore(t *testing.T, onChange func(core.Event)) (*core.Supervisor, *core.Client) {
	t.Helper()
	if os.Getenv(core.BinaryEnv) == "" {
		t.Setenv(core.BinaryEnv, filepath.Join("..", "..", "core", core.BinaryName()))
	}
	binary, err := core.FindBinary()
	if err != nil {
		t.Skip(err)
	}

	dir := t.TempDir()
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(freePort(t)))
	cfg, err := config.Build(config.Options{
		RoutingMode: "rule",
		LogLevel:    "info",
		MixedPort:   freePort(t),
		Auth:        &config.Credentials{User: "ctun", Password: "test"},
		Controller:  config.Controller{Addr: addr, Secret: "secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(cfgPath, cfg, 0o600); err != nil {
		t.Fatal(err)
	}

	sup := core.NewSupervisor(onChange)
	client, err := sup.Start(context.Background(), core.Spec{
		Binary: binary, WorkDir: dir, ConfigPath: cfgPath,
		ControllerAddr: addr, ControllerSecret: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = sup.Stop(context.Background()) })
	return sup, client
}

func TestCoreLifecycle(t *testing.T) {
	var mu sync.Mutex
	var statuses []core.Status
	sup, client := startCore(t, func(e core.Event) {
		mu.Lock()
		statuses = append(statuses, e.Status)
		mu.Unlock()
		if e.Err != nil {
			t.Errorf("unexpected error event: %v", e.Err)
		}
	})

	if v, err := client.Version(context.Background()); err != nil || v == "" {
		t.Fatalf("Version() = %q, %v", v, err)
	}
	if err := client.SetMode(context.Background(), "global"); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	got := 0
	err := client.StreamTraffic(ctx, func(core.Traffic) {
		if got++; got == 2 {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("StreamTraffic: %v (samples %d)", err, got)
	}

	if err := sup.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if sup.Status() != core.StatusStopped {
		t.Fatalf("status = %s", sup.Status())
	}
	mu.Lock()
	defer mu.Unlock()
	want := []core.Status{core.StatusStarting, core.StatusRunning, core.StatusStopping, core.StatusStopped}
	if len(statuses) != len(want) {
		t.Fatalf("statuses = %v, want %v", statuses, want)
	}
	for i := range want {
		if statuses[i] != want[i] {
			t.Fatalf("statuses = %v, want %v", statuses, want)
		}
	}
}

func TestCoreCrashIsReported(t *testing.T) {
	crashed := make(chan error, 1)
	sup, _ := startCore(t, func(e core.Event) {
		if e.Err != nil {
			crashed <- e.Err
		}
	})

	proc, err := os.FindProcess(sup.PID())
	if err != nil {
		t.Fatal(err)
	}
	if err := proc.Kill(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-crashed:
		t.Logf("reported: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("crash was not reported")
	}
}

func TestCoreRulesAPI(t *testing.T) {
	_, client := startCore(t, nil)
	ctx := context.Background()

	rules, err := client.Rules(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(rules) == 0 || rules[len(rules)-1].Type != "Match" || rules[0].Extra == nil {
		t.Fatalf("rules = %+v", rules)
	}
	if err := client.SetRuleDisabled(ctx, 0, true); err != nil {
		t.Fatal(err)
	}
	if rules, err = client.Rules(ctx); err != nil || !rules[0].Extra.Disabled {
		t.Fatalf("rule 0 not disabled: %+v, %v", rules, err)
	}
	if providers, err := client.RuleProviders(ctx); err != nil || len(providers) != 0 {
		t.Fatalf("providers = %v, %v", providers, err)
	}
	if err := client.UpdateRuleProvider(ctx, "missing"); err == nil {
		t.Fatal("updating an unknown rule set must fail")
	}
}
