package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Status string

const (
	StatusStopped  Status = "stopped"
	StatusStarting Status = "starting"
	StatusRunning  Status = "running"
	StatusStopping Status = "stopping"
)

const (
	readyTimeout  = 10 * time.Second
	readyInterval = 100 * time.Millisecond
	stopTimeout   = 5 * time.Second
	logLimit      = 2000
	errorTail     = 8
)

var ErrAlreadyRunning = errors.New("core: already running")

type Spec struct {
	Binary           string
	WorkDir          string
	ConfigPath       string
	ControllerAddr   string
	ControllerSecret string
}

type Event struct {
	Status Status
	Err    error
}

type Supervisor struct {
	onChange func(Event)
	logs     *LogBuffer

	mu       sync.Mutex
	status   Status
	cmd      *exec.Cmd
	exited   chan struct{}
	stopping bool
}

func NewSupervisor(onChange func(Event)) *Supervisor {
	return &Supervisor{onChange: onChange, logs: NewLogBuffer(logLimit), status: StatusStopped}
}

func (s *Supervisor) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Supervisor) PID() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cmd == nil || s.cmd.Process == nil {
		return 0
	}
	return s.cmd.Process.Pid
}

func (s *Supervisor) Start(ctx context.Context, spec Spec) (*Client, error) {
	s.mu.Lock()
	if s.status != StatusStopped {
		s.mu.Unlock()
		return nil, ErrAlreadyRunning
	}
	s.status = StatusStarting
	s.mu.Unlock()
	s.notify(Event{Status: StatusStarting})

	client, err := s.launch(ctx, spec)
	if err != nil {
		s.setStopped(Event{Status: StatusStopped, Err: err})
		return nil, err
	}

	s.mu.Lock()
	s.status = StatusRunning
	s.mu.Unlock()
	s.notify(Event{Status: StatusRunning})
	return client, nil
}

func (s *Supervisor) launch(ctx context.Context, spec Spec) (*Client, error) {
	if out, err := exec.CommandContext(ctx, spec.Binary, "-t", "-d", spec.WorkDir, "-f", spec.ConfigPath).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("core: config rejected: %s", lastLines(string(out), errorTail))
	}

	// The process outlives Start's context; Stop owns its lifetime.
	cmd := exec.CommandContext(context.WithoutCancel(ctx), spec.Binary, "-d", spec.WorkDir, "-f", spec.ConfigPath)
	cmd.Stdout = s.logs
	cmd.Stderr = s.logs
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("core: start %s: %w", spec.Binary, err)
	}

	exited := make(chan struct{})
	s.mu.Lock()
	s.cmd, s.exited, s.stopping = cmd, exited, false
	s.mu.Unlock()
	go s.wait(cmd, exited, spec.WorkDir)
	if err := writePID(spec.WorkDir, cmd.Process.Pid); err != nil {
		_ = s.Stop(context.Background())
		return nil, err
	}

	client := NewClient(spec.ControllerAddr, spec.ControllerSecret)
	if err := WaitReady(ctx, client, exited); err != nil {
		_ = s.Stop(context.Background())
		return nil, err
	}
	return client, nil
}

func WaitReady(ctx context.Context, client *Client, exited <-chan struct{}) error {
	ctx, cancel := context.WithTimeout(ctx, readyTimeout)
	defer cancel()
	ticker := time.NewTicker(readyInterval)
	defer ticker.Stop()
	for {
		if _, err := client.Version(ctx); err == nil {
			return nil
		}
		select {
		case <-exited:
			return errors.New("core: exited during startup")
		case <-ctx.Done():
			return fmt.Errorf("core: API did not come up: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

func (s *Supervisor) wait(cmd *exec.Cmd, exited chan struct{}, workDir string) {
	err := cmd.Wait()
	removePID(workDir)
	close(exited)

	s.mu.Lock()
	if s.cmd != cmd {
		s.mu.Unlock()
		return
	}
	expected := s.stopping || s.status == StatusStarting
	s.mu.Unlock()

	event := Event{Status: StatusStopped}
	if !expected {
		event.Err = fmt.Errorf("core: exited unexpectedly (%w): %s", err, strings.Join(s.logs.Tail(errorTail), "\n"))
	}
	s.setStopped(event)
}

func (s *Supervisor) Stop(ctx context.Context) error {
	s.mu.Lock()
	cmd, exited := s.cmd, s.exited
	if cmd == nil {
		s.mu.Unlock()
		return nil
	}
	s.stopping = true
	wasRunning := s.status == StatusRunning
	if wasRunning {
		s.status = StatusStopping
	}
	s.mu.Unlock()
	if wasRunning {
		s.notify(Event{Status: StatusStopping})
	}

	_ = cmd.Process.Signal(os.Interrupt)
	select {
	case <-exited:
		return nil
	case <-time.After(stopTimeout):
	case <-ctx.Done():
	}
	if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("core: kill: %w", err)
	}
	<-exited
	return nil
}

func (s *Supervisor) setStopped(event Event) {
	s.mu.Lock()
	if s.status == StatusStopped && event.Err == nil {
		s.mu.Unlock()
		return
	}
	s.status, s.cmd, s.exited, s.stopping = StatusStopped, nil, nil, false
	s.mu.Unlock()
	s.notify(event)
}

func (s *Supervisor) notify(event Event) {
	if s.onChange != nil {
		s.onChange(event)
	}
}

func lastLines(text string, n int) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	return strings.Join(lines[max(0, len(lines)-n):], "\n")
}
