//go:build unix

package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const pidFile = "core.pid"

func writePID(workDir string, pid int) error {
	if err := os.WriteFile(filepath.Join(workDir, pidFile), []byte(strconv.Itoa(pid)), 0o600); err != nil {
		return fmt.Errorf("core: write pid: %w", err)
	}
	return nil
}

func removePID(workDir string) {
	_ = os.Remove(filepath.Join(workDir, pidFile))
}

// StopOrphan stops a core left running by a crashed app (it would keep the proxy port open)
// and reports whether there was one. The PID is trusted only while its command line still
// points at workDir.
func StopOrphan(ctx context.Context, workDir string) (bool, error) {
	data, err := os.ReadFile(filepath.Join(workDir, pidFile))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("core: read pid: %w", err)
	}
	defer removePID(workDir)
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 1 {
		return false, nil //nolint:nilerr // a garbled pid file just gets removed
	}
	out, err := exec.CommandContext(ctx, "ps", "-p", strconv.Itoa(pid), "-o", "command=").Output()
	if err != nil || !strings.Contains(string(out), "-d "+workDir) {
		return false, nil //nolint:nilerr // no such process, or not our core
	}
	if err := syscall.Kill(pid, syscall.SIGINT); err != nil {
		return false, nil //nolint:nilerr // already gone
	}
	deadline := time.Now().Add(stopTimeout)
	for time.Now().Before(deadline) {
		if syscall.Kill(pid, 0) != nil {
			return true, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err := syscall.Kill(pid, syscall.SIGKILL); err != nil && !errors.Is(err, syscall.ESRCH) {
		return true, fmt.Errorf("core: kill orphan: %w", err)
	}
	return true, nil
}
