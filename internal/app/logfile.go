package app

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	logFileName = "ctun.log"
	maxLogSize  = 5 << 20
)

func OpenLogFile(dir string) (*os.File, error) {
	path := filepath.Join(dir, logFileName)
	switch info, err := os.Stat(path); {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, fmt.Errorf("stat log: %w", err)
	case info.Size() > maxLogSize:
		if err := os.Rename(path, path+".1"); err != nil {
			return nil, fmt.Errorf("rotate log: %w", err)
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open log: %w", err)
	}
	return f, nil
}
