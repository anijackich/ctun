//go:build unix

package core

import (
	"os/exec"
	"testing"
	"time"
)

func startFake(t *testing.T, dir string) (int, <-chan struct{}) {
	t.Helper()
	cmd := exec.CommandContext(t.Context(), "sh", "-c", "trap 'exit 0' INT; sleep 10 & wait", "-d", dir)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { _ = cmd.Wait(); close(done) }()
	t.Cleanup(func() { _ = cmd.Process.Kill(); <-done })
	return cmd.Process.Pid, done
}

func exited(done <-chan struct{}, within time.Duration) bool {
	select {
	case <-done:
		return true
	case <-time.After(within):
		return false
	}
}

func TestStopOrphanStopsOnlyOurCore(t *testing.T) {
	dir := t.TempDir()
	pid, done := startFake(t, dir)
	if err := writePID(dir, pid); err != nil {
		t.Fatal(err)
	}
	if stopped, err := StopOrphan(t.Context(), dir); err != nil || !stopped {
		t.Fatalf("stopped = %v, err = %v", stopped, err)
	}
	if !exited(done, 3*time.Second) {
		t.Fatal("orphaned core still running")
	}

	pid, done = startFake(t, t.TempDir())
	if err := writePID(dir, pid); err != nil {
		t.Fatal(err)
	}
	if stopped, err := StopOrphan(t.Context(), dir); err != nil || stopped {
		t.Fatalf("stopped = %v, err = %v", stopped, err)
	}
	if exited(done, 300*time.Millisecond) {
		t.Fatal("a process from another directory must be left alone")
	}
}
