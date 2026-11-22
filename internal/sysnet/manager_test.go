package sysnet

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

type fakeBackend struct {
	current string
	applied int
}

func (f *fakeBackend) snapshot(context.Context) (json.RawMessage, error) {
	data, err := json.Marshal(f.current)
	if err != nil {
		return nil, fmt.Errorf("encode: %w", err)
	}
	return data, nil
}

func (f *fakeBackend) apply(_ context.Context, value string) error {
	f.applied++
	f.current = value
	return nil
}

func (f *fakeBackend) restore(_ context.Context, snapshot json.RawMessage) error {
	if err := json.Unmarshal(snapshot, &f.current); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
}

func TestEnableDisableRestoresUserSetting(t *testing.T) {
	fake := &fakeBackend{current: "user"}
	m := &Manager[string]{statePath: filepath.Join(t.TempDir(), "state.json"), backend: fake}
	ctx := t.Context()

	if err := m.Enable(ctx, "ours"); err != nil {
		t.Fatal(err)
	}

	if err := m.Enable(ctx, "ours again"); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(m.statePath); err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("snapshot file: %v %v", info, err)
	}
	if restored, err := m.Disable(ctx); err != nil || !restored {
		t.Fatalf("restored = %v, err = %v", restored, err)
	}
	if fake.current != "user" || fake.applied != 2 {
		t.Fatalf("current = %q, applied = %d", fake.current, fake.applied)
	}
	if restored, err := m.Disable(ctx); err != nil || restored {
		t.Fatalf("second disable: restored = %v, err = %v", restored, err)
	}
}

func TestDisableUndoesCrashedRun(t *testing.T) {
	state := filepath.Join(t.TempDir(), "state.json")
	crashed := &Manager[string]{statePath: state, backend: &fakeBackend{current: "user"}}
	if err := crashed.Enable(t.Context(), "ours"); err != nil {
		t.Fatal(err)
	}
	fresh := &fakeBackend{current: "ours"}
	if _, err := (&Manager[string]{statePath: state, backend: fresh}).Disable(t.Context()); err != nil {
		t.Fatal(err)
	}
	if fresh.current != "user" {
		t.Fatalf("current = %q after recovery", fresh.current)
	}
}

func TestUnsupportedPlatform(t *testing.T) {
	m := &Manager[string]{statePath: filepath.Join(t.TempDir(), "x")}
	restored, err := m.Disable(t.Context())
	if m.Supported() || !errors.Is(m.Enable(t.Context(), "v"), ErrUnsupported) || restored || err != nil {
		t.Fatal("a manager without backend must refuse to enable and ignore disable")
	}
}
