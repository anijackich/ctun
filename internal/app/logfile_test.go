package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenLogFileRotatesLargeLogs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, logFileName)
	if err := os.WriteFile(path, make([]byte, maxLogSize+1), 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := OpenLogFile(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if info, err := os.Stat(path); err != nil || info.Size() != 0 {
		t.Fatalf("fresh log: %v, %v", info, err)
	}
	if info, err := os.Stat(path + ".1"); err != nil || info.Size() != maxLogSize+1 {
		t.Fatalf("rotated log: %v, %v", info, err)
	}
}
