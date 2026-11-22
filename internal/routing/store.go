package routing

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"sync"

	"ctun/internal/fsutil"
)

var (
	ErrNotFound   = errors.New("routing: profile not found")
	ErrLastOne    = errors.New("routing: the last profile cannot be removed")
	ErrNameNeeded = errors.New("routing: profile name is empty")
)

type index struct {
	Active   string    `json:"active"`
	Profiles []Profile `json:"profiles"`
}

type Store struct {
	path string

	mu  sync.Mutex
	idx index
}

func Open(path string) (*Store, error) {
	s := &Store{path: path}
	data, err := os.ReadFile(path)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, fmt.Errorf("routing: read: %w", err)
	default:
		if err := json.Unmarshal(data, &s.idx); err != nil {
			return nil, fmt.Errorf("routing: parse: %w", err)
		}
	}
	s.idx = s.idx.normalize()
	return s, nil
}

func (idx index) normalize() index {
	profiles := make([]Profile, 0, len(idx.Profiles))
	for _, p := range idx.Profiles {
		if p.ID != "" && !slices.ContainsFunc(profiles, func(q Profile) bool { return q.ID == p.ID }) {
			profiles = append(profiles, p.normalize())
		}
	}
	if len(profiles) == 0 {
		profiles = []Profile{Default()}
	}
	idx.Profiles = profiles
	if !slices.ContainsFunc(profiles, func(p Profile) bool { return p.ID == idx.Active }) {
		idx.Active = profiles[0].ID
	}
	return idx
}

func (s *Store) List() []Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.idx.Profiles)
}

func (s *Store) ActiveID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.idx.Active
}

func (s *Store) Active() Profile {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.idx.Profiles[s.find(s.idx.Active)]
}

func (s *Store) Save(p Profile) (Profile, error) {
	p = p.normalize()
	if p.Name == "" {
		return Profile{}, ErrNameNeeded
	}
	return p, s.update(func(idx *index) error {
		i := slices.IndexFunc(idx.Profiles, func(q Profile) bool { return q.ID == p.ID })
		if i < 0 {
			return ErrNotFound
		}
		idx.Profiles[i] = p
		return nil
	})
}

func (s *Store) Create(name, from string) (Profile, error) {
	id, err := newID()
	if err != nil {
		return Profile{}, err
	}
	p := Profile{Name: name, Final: Proxy}
	err = s.update(func(idx *index) error {
		if from != "" {
			i := slices.IndexFunc(idx.Profiles, func(q Profile) bool { return q.ID == from })
			if i < 0 {
				return ErrNotFound
			}
			p = idx.Profiles[i]
			p.Lists = cloneLists(p.Lists)
			p.Name = name
		}
		p.ID = id
		p = p.normalize()
		if p.Name == "" {
			return ErrNameNeeded
		}
		idx.Profiles = append(idx.Profiles, p)
		return nil
	})
	return p, err
}

func (s *Store) Remove(id string) error {
	return s.update(func(idx *index) error {
		if len(idx.Profiles) == 1 {
			return ErrLastOne
		}
		i := slices.IndexFunc(idx.Profiles, func(q Profile) bool { return q.ID == id })
		if i < 0 {
			return ErrNotFound
		}
		idx.Profiles = slices.Delete(idx.Profiles, i, i+1)
		if idx.Active == id {
			idx.Active = idx.Profiles[0].ID
		}
		return nil
	})
}

func (s *Store) SetActive(id string) error {
	return s.update(func(idx *index) error {
		if !slices.ContainsFunc(idx.Profiles, func(q Profile) bool { return q.ID == id }) {
			return ErrNotFound
		}
		idx.Active = id
		return nil
	})
}

func (s *Store) update(fn func(*index) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	next := index{Active: s.idx.Active, Profiles: slices.Clone(s.idx.Profiles)}
	if err := fn(&next); err != nil {
		return err
	}
	data, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return fmt.Errorf("routing: encode: %w", err)
	}
	if err := fsutil.WriteFileAtomic(s.path, data, 0o600); err != nil {
		return fmt.Errorf("routing: save: %w", err)
	}
	s.idx = next
	return nil
}

func (s *Store) find(id string) int {
	return max(0, slices.IndexFunc(s.idx.Profiles, func(p Profile) bool { return p.ID == id }))
}

func cloneLists(lists []List) []List {
	out := make([]List, len(lists))
	for i, l := range lists {
		out[i] = List{Target: l.Target, Entries: slices.Clone(l.Entries)}
	}
	return out
}

func newID() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("routing: generate id: %w", err)
	}
	return hex.EncodeToString(b), nil
}
