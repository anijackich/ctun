package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return NewClient(strings.TrimPrefix(srv.URL, "http://"), "secret")
}

func TestClientSendsBearerSecret(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer secret" {
			http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
			return
		}
		fmt.Fprint(w, `{"meta":true,"version":"v1.19.34"}`)
	})
	v, err := c.Version(context.Background())
	if err != nil || v != "v1.19.34" {
		t.Fatalf("Version() = %q, %v", v, err)
	}
}

func TestClientReportsHTTPErrors(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, `{"message":"Unauthorized"}`, http.StatusUnauthorized)
	})
	_, err := c.Version(context.Background())
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("err = %v, want 401", err)
	}
}

func TestSetModePatchesConfigs(t *testing.T) {
	var got map[string]string
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/configs" {
			http.Error(w, "unexpected", http.StatusBadRequest)
			return
		}
		_ = json.NewDecoder(r.Body).Decode(&got)
		w.WriteHeader(http.StatusNoContent)
	})
	if err := c.SetMode(context.Background(), "global"); err != nil {
		t.Fatal(err)
	}
	if got["mode"] != "global" {
		t.Fatalf("body = %v", got)
	}
}

func TestStreamTrafficDecodesSamplesUntilCancel(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		flusher := w.(http.Flusher)
		for i := 1; ; i++ {
			fmt.Fprintf(w, `{"up":%d,"down":%d,"upTotal":%d,"downTotal":%d}`+"\n", i, i*10, i, i*10)
			flusher.Flush()
			select {
			case <-r.Context().Done():
				return
			case <-time.After(5 * time.Millisecond):
			}
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	var samples []Traffic
	err := c.StreamTraffic(ctx, func(s Traffic) {
		samples = append(samples, s)
		if len(samples) == 3 {
			cancel()
		}
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v, want context.Canceled", err)
	}
	if samples[2].Down != 30 || samples[2].DownTotal != 30 {
		t.Fatalf("samples = %+v", samples)
	}
}
