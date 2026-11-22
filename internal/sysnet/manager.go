package sysnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"ctun/internal/fsutil"
)

var ErrUnsupported = errors.New("not supported on this platform")

type backend[T any] interface {
	snapshot(ctx context.Context) (json.RawMessage, error)
	apply(ctx context.Context, value T) error
	restore(ctx context.Context, snapshot json.RawMessage) error
}

type Manager[T any] struct {
	statePath string
	backend   backend[T]

	mu sync.Mutex
}

func (m *Manager[T]) Supported() bool {
	return m.backend != nil
}

func (m *Manager[T]) Enable(ctx context.Context, value T) error {
	if m.backend == nil {
		return ErrUnsupported
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, err := os.Stat(m.statePath); errors.Is(err, os.ErrNotExist) {
		snapshot, err := m.backend.snapshot(ctx)
		if err != nil {
			return fmt.Errorf("read current setting: %w", err)
		}
		if err := fsutil.WriteFileAtomic(m.statePath, snapshot, 0o600); err != nil {
			return fmt.Errorf("save current setting: %w", err)
		}
	}
	if err := m.backend.apply(ctx, value); err != nil {
		return fmt.Errorf("apply setting: %w", err)
	}
	return nil
}

func (m *Manager[T]) Disable(ctx context.Context) (bool, error) {
	if m.backend == nil {
		return false, nil
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot, err := os.ReadFile(m.statePath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read saved setting: %w", err)
	}
	if err := m.backend.restore(ctx, snapshot); err != nil {
		return false, fmt.Errorf("restore setting: %w", err)
	}
	if err := os.Remove(m.statePath); err != nil {
		return true, fmt.Errorf("forget saved setting: %w", err)
	}
	return true, nil
}
