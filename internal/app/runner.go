package app

import (
	"context"
	"fmt"
	"path/filepath"

	"ctun/internal/core"
	"ctun/internal/fsutil"
)

type runner interface {
	Start(ctx context.Context, cfg []byte, controllerAddr, secret string) (*core.Client, error)
	Stop(ctx context.Context) error
	Status() core.Status

	Stage(cfg []byte) (string, error)
}

type localRunner struct {
	sup    *core.Supervisor
	binary func() (string, error)
	dir    string
}

func newLocalRunner(dir string, binary func() (string, error), onChange func(core.Event)) *localRunner {
	return &localRunner{sup: core.NewSupervisor(onChange), binary: binary, dir: dir}
}

func (r *localRunner) Start(ctx context.Context, cfg []byte, controllerAddr, secret string) (*core.Client, error) {
	binary, err := r.binary()
	if err != nil {
		return nil, err
	}
	path, err := r.Stage(cfg)
	if err != nil {
		return nil, err
	}
	return r.sup.Start(ctx, core.Spec{
		Binary:           binary,
		WorkDir:          r.dir,
		ConfigPath:       path,
		ControllerAddr:   controllerAddr,
		ControllerSecret: secret,
	})
}

func (r *localRunner) Stop(ctx context.Context) error { return r.sup.Stop(ctx) }

func (r *localRunner) Status() core.Status { return r.sup.Status() }

func (r *localRunner) Stage(cfg []byte) (string, error) {
	path := filepath.Join(r.dir, "config.yaml")
	if err := fsutil.WriteFileAtomic(path, cfg, 0o600); err != nil {
		return "", fmt.Errorf("write core config: %w", err)
	}
	return path, nil
}
