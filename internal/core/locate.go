package core

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const BinaryEnv = "CTUN_CORE"

var ErrBinaryNotFound = errors.New("core: mihomo binary not found (run `task core:fetch` or set " + BinaryEnv + ")")

func BinaryName() string {
	name := "mihomo-" + runtime.GOOS + "-" + runtime.GOARCH
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func FindBinary() (string, error) {
	if path := os.Getenv(BinaryEnv); path != "" {
		return checkExecutable(path)
	}

	exe, err := os.Executable()
	if err != nil {
		return "", ErrBinaryNotFound
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	dir := filepath.Dir(exe)
	candidates := []string{
		filepath.Join(dir, BinaryName()),
		filepath.Join(dir, "..", "Resources", BinaryName()),
	}
	for d := dir; ; d = filepath.Dir(d) {
		candidates = append(candidates, filepath.Join(d, "core", BinaryName()))
		if parent := filepath.Dir(d); parent == d {
			break
		}
	}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "core", BinaryName()))
	}

	for _, path := range candidates {
		if found, err := checkExecutable(path); err == nil {
			return found, nil
		}
	}
	return "", ErrBinaryNotFound
}

func checkExecutable(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() || (runtime.GOOS != "windows" && info.Mode()&0o111 == 0) {
		return "", ErrBinaryNotFound
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("core: resolve %s: %w", path, err)
	}
	return abs, nil
}
