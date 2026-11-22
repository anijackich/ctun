package profile

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"ctun/internal/config"
	"ctun/internal/fsutil"
)

var ErrNotFound = errors.New("profile: not found")

type index struct {
	Active   string    `json:"active"`
	Profiles []Profile `json:"profiles"`
}

type Store struct {
	dir string

	mu    sync.Mutex
	index index
}

func OpenStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("profile: create store: %w", err)
	}
	s := &Store{dir: dir}
	data, err := os.ReadFile(s.indexPath())
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, fmt.Errorf("profile: read index: %w", err)
	default:
		if err := json.Unmarshal(data, &s.index); err != nil {
			return nil, fmt.Errorf("profile: parse index: %w", err)
		}
	}
	return s, nil
}

func (s *Store) indexPath() string           { return filepath.Join(s.dir, "profiles.json") }
func (s *Store) sourcePath(id string) string { return filepath.Join(s.dir, id+".sub") }

func (s *Store) List() []Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.index.Profiles)
}

func (s *Store) ActiveID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.index.Active
}

func (s *Store) Get(id string) (Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i := s.index.find(id); i >= 0 {
		return s.index.Profiles[i], nil
	}
	return Profile{}, ErrNotFound
}

func (s *Store) Layer(id string) (*config.Profile, error) {
	body, err := os.ReadFile(s.sourcePath(id))
	if err != nil {
		return nil, fmt.Errorf("profile: read subscription: %w", err)
	}
	return Parse(body)
}

func (s *Store) Put(p Profile, body []byte) error {
	if body != nil {
		if err := fsutil.WriteFileAtomic(s.sourcePath(p.ID), body, 0o600); err != nil {
			return fmt.Errorf("profile: save subscription: %w", err)
		}
	}
	return s.mutate(func(idx *index) error {
		if i := idx.find(p.ID); i >= 0 {
			idx.Profiles[i] = p
		} else {
			idx.Profiles = append(idx.Profiles, p)
		}
		if idx.Active == "" {
			idx.Active = p.ID
		}
		return nil
	})
}

func (s *Store) Update(id string, fn func(*Profile)) (Profile, error) {
	var updated Profile
	err := s.mutate(func(idx *index) error {
		i := idx.find(id)
		if i < 0 {
			return ErrNotFound
		}
		fn(&idx.Profiles[i])
		updated = idx.Profiles[i]
		return nil
	})
	return updated, err
}

func (s *Store) SetActive(id string) error {
	return s.mutate(func(idx *index) error {
		if idx.find(id) < 0 {
			return ErrNotFound
		}
		idx.Active = id
		return nil
	})
}

func (s *Store) Remove(id string) error {
	err := s.mutate(func(idx *index) error {
		i := idx.find(id)
		if i < 0 {
			return ErrNotFound
		}
		idx.Profiles = slices.Delete(idx.Profiles, i, i+1)
		if idx.Active == id {
			idx.Active = ""
			if len(idx.Profiles) > 0 {
				idx.Active = idx.Profiles[min(i, len(idx.Profiles)-1)].ID
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	if err := os.Remove(s.sourcePath(id)); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("profile: remove subscription: %w", err)
	}
	return nil
}

func (idx *index) find(id string) int {
	return slices.IndexFunc(idx.Profiles, func(p Profile) bool { return p.ID == id })
}

func (s *Store) mutate(fn func(*index) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := index{Active: s.index.Active, Profiles: slices.Clone(s.index.Profiles)}
	if err := fn(&next); err != nil {
		return err
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err == nil {
		err = fsutil.WriteFileAtomic(s.indexPath(), data, 0o600)
	}
	if err != nil {
		return fmt.Errorf("profile: save index: %w", err)
	}
	s.index = next
	return nil
}
