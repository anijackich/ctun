package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Paths struct {
	Data     string
	Runtime  string
	Profiles string
	Settings string
	Routing  string
	Logs     string
}

func NewPaths() (Paths, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return Paths{}, fmt.Errorf("locate config dir: %w", err)
	}
	data := filepath.Join(base, "ctun")
	logs, err := logDir()
	if err != nil {
		return Paths{}, err
	}
	p := Paths{
		Data:     data,
		Runtime:  filepath.Join(data, "runtime"),
		Profiles: filepath.Join(data, "profiles"),
		Settings: filepath.Join(data, "settings.json"),
		Routing:  filepath.Join(data, "routing.json"),
		Logs:     logs,
	}
	for _, dir := range []string{p.Data, p.Runtime, p.Profiles, p.Logs} {
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return Paths{}, fmt.Errorf("create %s: %w", dir, err)
		}
	}
	return p, nil
}

func logDir() (string, error) {
	if runtime.GOOS == "darwin" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("locate home dir: %w", err)
		}
		return filepath.Join(home, "Library", "Logs", "ctun"), nil
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return "", fmt.Errorf("locate cache dir: %w", err)
	}
	return filepath.Join(cache, "ctun", "logs"), nil
}
