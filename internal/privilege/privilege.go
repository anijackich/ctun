package privilege

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"ctun/internal/helper"
)

type State string

const (
	StateInstalled   State = "installed"
	StateMissing     State = "missing"
	StateOutdated    State = "outdated"
	StateUnsupported State = "unsupported"
)

const HelperEnv = "CTUN_HELPER"

var ErrHelperNotFound = errors.New("privilege: ctun-helper binary not found (build it or set " + HelperEnv + ")")

func FindHelper() (string, error) {
	if path := os.Getenv(HelperEnv); path != "" {
		return absolute(path)
	}
	exe, err := os.Executable()
	if err != nil {
		return "", ErrHelperNotFound
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	name := helper.HelperName
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	dir := filepath.Dir(exe)
	candidates := []string{filepath.Join(dir, name), filepath.Join(dir, "..", "Resources", name)}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "bin", name))
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return absolute(candidate)
		}
	}
	return "", ErrHelperNotFound
}

func absolute(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("privilege: resolve %s: %w", path, err)
	}
	return abs, nil
}
