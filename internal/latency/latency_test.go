package latency

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"ctun/internal/config"
)

func TestRunReportsEachResultAndFailuresAsZero(t *testing.T) {
	delays := map[string]int{"a": 10, "b": 20}
	var mu sync.Mutex
	got := map[string]int{}
	Run(context.Background(), []string{"a", "b", "broken"}, func(_ context.Context, name string) (int, error) {
		if d, ok := delays[name]; ok {
			return d, nil
		}
		return 0, errors.New("down")
	}, func(name string, delay int) {
		mu.Lock()
		got[name] = delay
		mu.Unlock()
	})
	if len(got) != 3 || got["a"] != 10 || got["b"] != 20 || got["broken"] != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestRunStreamsWithoutWaitingForSlowOnes(t *testing.T) {
	results := make(chan string, 2)
	done := make(chan struct{})
	go func() {
		Run(context.Background(), []string{"fast", "slow"}, func(ctx context.Context, name string) (int, error) {
			if name == "slow" {
				<-ctx.Done()
				return 0, ctx.Err()
			}
			return 1, nil
		}, func(name string, _ int) { results <- name })
		close(done)
	}()
	select {
	case name := <-results:
		if name != "fast" {
			t.Fatalf("first result = %s, want fast", name)
		}
	case <-time.After(time.Second):
		t.Fatal("fast result was held back by the slow one")
	}
	<-done
}

func TestDirectThroughLocalHTTPProxy(t *testing.T) {
	var hits sync.WaitGroup
	hits.Add(1)
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodConnect {
			http.Error(w, "connect only", http.StatusMethodNotAllowed)
			return
		}
		defer hits.Done()

		hj, _ := w.(http.Hijacker)
		conn, buf, err := hj.Hijack()
		if err != nil {
			return
		}
		defer conn.Close()
		_, _ = buf.WriteString("HTTP/1.1 200 Connection established\r\n\r\n")
		_ = buf.Flush()
		_ = conn.SetDeadline(time.Now().Add(2 * time.Second))
		b := make([]byte, 1)
		_, _ = conn.Read(b)
	}))
	defer proxy.Close()

	host, port, _ := net.SplitHostPort(proxy.Listener.Addr().String())
	p, _ := strconv.Atoi(port)
	test := Direct([]map[string]any{{"name": "local", "type": "http", "server": host, "port": p}}, config.DefaultNameservers)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, _ = test(ctx, "local")
	waitOrFail(t, &hits)

	if _, err := test(ctx, "missing"); err == nil {
		t.Fatal("unknown proxy must fail")
	}
}

func waitOrFail(t *testing.T, wg *sync.WaitGroup) {
	t.Helper()
	done := make(chan struct{})
	go func() { wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("proxy was never dialled")
	}
}

func TestPhysicalInterfaceSkipsLoopbackAndTunnels(t *testing.T) {
	name := physicalInterface()
	if name == "" {
		t.Skip("no physical interface on this machine")
	}
	iface, err := net.InterfaceByName(name)
	if err != nil {
		t.Fatal(err)
	}
	if iface.Flags&(net.FlagLoopback|net.FlagPointToPoint) != 0 {
		t.Fatalf("%s is loopback or a tunnel (flags %v)", name, iface.Flags)
	}
	t.Logf("physical uplink: %s", name)
}
