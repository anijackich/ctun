package app

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"
)

func TestOSSyncAppliesChangesOnceAndRestoresOnShutdown(t *testing.T) {
	var mu sync.Mutex
	want, fail := false, false
	var applied []bool
	s := newOSSync(
		func() bool { mu.Lock(); defer mu.Unlock(); return want },
		func(_ context.Context, v bool) error {
			mu.Lock()
			defer mu.Unlock()
			if fail {
				fail = false
				return errors.New("busy")
			}
			applied = append(applied, v)
			return nil
		},
		func(error) {},
	)
	snapshot := func() []bool { mu.Lock(); defer mu.Unlock(); return slices.Clone(applied) }
	waitFor := func(n int) {
		t.Helper()
		deadline := time.Now().Add(2 * time.Second)
		for len(snapshot()) < n {
			if time.Now().After(deadline) {
				t.Fatalf("applied = %v, want %d calls", snapshot(), n)
			}
			time.Sleep(5 * time.Millisecond)
		}
	}

	s.start()
	waitFor(1)
	s.trigger()
	time.Sleep(50 * time.Millisecond)
	if got := snapshot(); len(got) != 1 {
		t.Fatalf("unchanged state re-applied: %v", got)
	}

	mu.Lock()
	want, fail = true, true
	mu.Unlock()
	s.trigger()
	time.Sleep(50 * time.Millisecond)
	s.trigger()
	waitFor(2)

	if err := s.shutdown(); err != nil {
		t.Fatal(err)
	}
	if got := snapshot(); !slices.Equal(got, []bool{false, true, false}) {
		t.Fatalf("applied = %v", got)
	}
}
