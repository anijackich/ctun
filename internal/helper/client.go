package helper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"ctun/internal/core"
)

const dialTimeout = 3 * time.Second

var ErrNotInstalled = errors.New("helper: not installed")

func dial(ctx context.Context, socket string) (net.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, dialTimeout)
	defer cancel()
	var d net.Dialer
	conn, err := d.DialContext(ctx, "unix", socket)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrNotInstalled, err)
	}
	return conn, nil
}

func call(ctx context.Context, socket string, req Request) (Response, error) {
	conn, err := dial(ctx, socket)
	if err != nil {
		return Response{}, err
	}
	defer conn.Close()
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}
	if err := send(conn, req); err != nil {
		return Response{}, err
	}
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return Response{}, fmt.Errorf("helper: read: %w", err)
	}
	if !resp.OK {
		return resp, fmt.Errorf("helper: %s", resp.Error)
	}
	return resp, nil
}

func Status(ctx context.Context, socket string) (Response, error) {
	return call(ctx, socket, Request{Op: OpStatus})
}

type Runner struct {
	Socket   string
	OnChange func(core.Event)

	mu       sync.Mutex
	status   core.Status
	conn     net.Conn
	done     chan struct{}
	stopping bool
}

func (r *Runner) Status() core.Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.status == "" {
		return core.StatusStopped
	}
	return r.status
}

func (r *Runner) Start(ctx context.Context, cfg []byte, controllerAddr, secret string) (*core.Client, error) {
	r.mu.Lock()
	if r.conn != nil {
		r.mu.Unlock()
		return nil, core.ErrAlreadyRunning
	}
	r.status = core.StatusStarting
	r.mu.Unlock()
	r.notify(core.Event{Status: core.StatusStarting})

	conn, err := r.open(ctx, cfg)
	if err != nil {
		r.finish(err)
		return nil, err
	}

	done := make(chan struct{})
	r.mu.Lock()
	r.conn, r.done, r.stopping = conn, done, false
	r.mu.Unlock()
	go r.watch(conn, done)

	client := core.NewClient(controllerAddr, secret)
	if err := core.WaitReady(ctx, client, done); err != nil {
		_ = r.Stop(context.Background())
		r.notify(core.Event{Status: core.StatusStopped, Err: err})
		return nil, err
	}
	r.mu.Lock()
	r.status = core.StatusRunning
	r.mu.Unlock()
	r.notify(core.Event{Status: core.StatusRunning})
	return client, nil
}

func (r *Runner) open(ctx context.Context, cfg []byte) (net.Conn, error) {
	conn, err := dial(ctx, r.Socket)
	if err != nil {
		return nil, err
	}
	if err := send(conn, Request{Op: OpRun, Config: cfg}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("helper: read: %w", err)
	}
	if !resp.OK {
		_ = conn.Close()
		return nil, fmt.Errorf("helper: %s", resp.Error)
	}
	return conn, nil
}

func (r *Runner) watch(conn net.Conn, done chan struct{}) {
	var resp Response
	err := json.NewDecoder(conn).Decode(&resp)
	close(done)

	r.mu.Lock()
	expected := r.stopping
	r.mu.Unlock()
	switch {
	case expected:
		r.finish(nil)
	case err != nil:
		r.finish(fmt.Errorf("helper: session lost: %w", err))
	default:
		r.finish(fmt.Errorf("core: %s: %s", resp.Error, strings.Join(resp.Tail, "\n")))
	}
}

func (r *Runner) Stop(ctx context.Context) error {
	r.mu.Lock()
	conn, done := r.conn, r.done
	if conn == nil {
		r.mu.Unlock()
		return nil
	}
	wasRunning := r.status == core.StatusRunning
	r.stopping = true
	if wasRunning {
		r.status = core.StatusStopping
	}
	r.mu.Unlock()
	if wasRunning {
		r.notify(core.Event{Status: core.StatusStopping})
	}

	_ = send(conn, Request{Op: OpStop})
	select {
	case <-done:
	case <-ctx.Done():
		_ = conn.Close()
		<-done
	}
	return nil
}

func (r *Runner) Stage(cfg []byte) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	resp, err := call(ctx, r.Socket, Request{Op: OpStage, Config: cfg})
	return resp.Path, err
}

func (r *Runner) finish(err error) {
	r.mu.Lock()
	if r.conn != nil {
		_ = r.conn.Close()
	}
	r.conn, r.done, r.stopping, r.status = nil, nil, false, core.StatusStopped
	r.mu.Unlock()
	r.notify(core.Event{Status: core.StatusStopped, Err: err})
}

func (r *Runner) notify(e core.Event) {
	if r.OnChange != nil {
		r.OnChange(e)
	}
}
