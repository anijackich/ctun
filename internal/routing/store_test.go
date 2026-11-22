package routing

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreLifecycle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "routing.json")
	s, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if s.Active().ID != defaultID || len(s.List()) != 1 {
		t.Fatalf("fresh store = %+v", s.List())
	}

	work, err := s.Create("Work", defaultID)
	if err != nil {
		t.Fatal(err)
	}
	work.Lists[0].Entries = []string{"geosite:category-ads-all"}
	if _, err := s.Save(work); err != nil {
		t.Fatal(err)
	}
	if def := s.List()[0]; len(def.Lists[0].Entries) != 0 {
		t.Fatal("a copy must not share lists with its source")
	}
	if err := s.SetActive(work.ID); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	if reopened.ActiveID() != work.ID || reopened.Active().Lists[0].Entries[0] != "geosite:category-ads-all" {
		t.Fatalf("reopened = %+v", reopened.List())
	}
	if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %v", info.Mode().Perm())
	}

	if err := reopened.Remove(work.ID); err != nil {
		t.Fatal(err)
	}
	if reopened.ActiveID() != defaultID {
		t.Fatal("removing the active profile must fall back to another")
	}
	if err := reopened.Remove(defaultID); !errors.Is(err, ErrLastOne) {
		t.Fatalf("removing the last profile: %v", err)
	}
	if _, err := reopened.Save(Profile{ID: "missing", Name: "x"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("saving unknown profile: %v", err)
	}
	if _, err := reopened.Create("  ", ""); !errors.Is(err, ErrNameNeeded) {
		t.Fatalf("empty name: %v", err)
	}
}
