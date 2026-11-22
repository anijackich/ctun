//go:build integration && (darwin || linux)

package helper_test

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"ctun/internal/config"
	"ctun/internal/core"
	"ctun/internal/helper"
)

func freePort(t *testing.T) int {
	t.Helper()
	var lc net.ListenConfig
	l, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

type fixture struct {
	socket string
	server *helper.Server
	addr   string
	cfg    []byte
}

func newFixture(t *testing.T, allowed uint32) *fixture {
	t.Helper()
	binary, err := filepath.Abs(filepath.Join("..", "..", "core", "mihomo-"+runtime.GOOS+"-"+runtime.GOARCH))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(binary); err != nil {
		t.Skip("core binary missing; run task core:fetch")
	}
	// Unix socket paths are limited to ~104 bytes: keep it short.
	dir, err := os.MkdirTemp("/tmp", "ctun-helper-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	socket := filepath.Join(dir, "h.sock")
	var lc net.ListenConfig
	l, err := lc.Listen(context.Background(), "unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &helper.Server{Core: binary, WorkDir: filepath.Join(dir, "work"), Authorize: helper.AllowUID(allowed)}
	go func() { _ = server.Serve(l) }()
	t.Cleanup(func() {
		_ = l.Close()
		server.Shutdown()
	})

	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(freePort(t)))
	cfg, err := config.Build(config.Options{
		RoutingMode: "rule",
		LogLevel:    "info",
		Controller:  config.Controller{Addr: addr, Secret: "secret"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &fixture{socket: socket, server: server, addr: addr, cfg: cfg}
}

func coreAlive(addr string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()
	_, err := core.NewClient(addr, "secret").Version(ctx)
	return err == nil
}

func TestRunnerLifecycle(t *testing.T) {
	f := newFixture(t, uint32(os.Getuid()))
	var mu sync.Mutex
	var statuses []core.Status
	r := &helper.Runner{Socket: f.socket, OnChange: func(e core.Event) {
		mu.Lock()
		statuses = append(statuses, e.Status)
		mu.Unlock()
		if e.Err != nil {
			t.Errorf("unexpected error event: %v", e.Err)
		}
	}}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	client, err := r.Start(ctx, f.cfg, f.addr, "secret")
	if err != nil {
		t.Fatal(err)
	}

	status, err := helper.Status(ctx, f.socket)
	if err != nil || !status.Running || status.Version != helper.Version {
		t.Fatalf("status = %+v, %v", status, err)
	}
	if _, err := (&helper.Runner{Socket: f.socket}).Start(ctx, f.cfg, f.addr, "secret"); err == nil {
		t.Fatal("a second session must be refused while the core runs")
	}

	path, err := r.Stage(f.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Reload(ctx, path); err != nil {
		t.Fatalf("hot reload from staged config: %v", err)
	}

	if err := r.Stop(ctx); err != nil {
		t.Fatal(err)
	}
	if coreAlive(f.addr) {
		t.Fatal("core still answers after Stop")
	}
	mu.Lock()
	defer mu.Unlock()
	want := []core.Status{core.StatusStarting, core.StatusRunning, core.StatusStopping, core.StatusStopped}
	if strings.Join(toStrings(statuses), ",") != strings.Join(toStrings(want), ",") {
		t.Fatalf("statuses = %v, want %v", statuses, want)
	}
}

func TestDroppedSessionStopsCore(t *testing.T) {
	f := newFixture(t, uint32(os.Getuid()))
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", f.socket)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.NewEncoder(conn).Encode(helper.Request{Op: helper.OpRun, Config: f.cfg}); err != nil {
		t.Fatal(err)
	}
	var resp helper.Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil || !resp.OK {
		t.Fatalf("run = %+v, %v", resp, err)
	}
	if err := core.WaitReady(ctx, core.NewClient(f.addr, "secret"), nil); err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()

	deadline := time.Now().Add(8 * time.Second)
	for coreAlive(f.addr) {
		if time.Now().After(deadline) {
			t.Fatal("core kept running after the session dropped")
		}
		time.Sleep(100 * time.Millisecond)
	}
	status, err := helper.Status(ctx, f.socket)
	if err != nil || status.Running {
		t.Fatalf("daemon must be idle again: %+v, %v", status, err)
	}
}

func TestOtherUsersAreRejected(t *testing.T) {
	f := newFixture(t, uint32(os.Getuid())+1)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := helper.Status(ctx, f.socket)
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("err = %v, want not allowed", err)
	}
}

func toStrings(list []core.Status) []string {
	out := make([]string, len(list))
	for i, s := range list {
		out[i] = string(s)
	}
	return out
}
