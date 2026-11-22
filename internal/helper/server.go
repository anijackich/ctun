package helper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"ctun/internal/core"
	"ctun/internal/fsutil"
)

const (
	stopTimeout = 5 * time.Second
	logLimit    = 500
	errorTail   = 8
)

type Server struct {
	Core      string
	WorkDir   string
	Authorize func(net.Conn) error

	mu      sync.Mutex
	running *session
}

type session struct {
	cmd      *exec.Cmd
	exited   chan struct{}
	logs     *core.LogBuffer
	stopOnce sync.Once
}

func (s *Server) Serve(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			return fmt.Errorf("helper: accept: %w", err)
		}
		go s.handle(conn)
	}
}

func (s *Server) Shutdown() {
	s.mu.Lock()
	sess := s.running
	s.mu.Unlock()
	if sess != nil {
		sess.stop()
	}
}

func (s *Server) handle(conn net.Conn) {
	if err := s.Authorize(conn); err != nil {
		slog.Warn("rejected connection", "err", err)
		_ = send(conn, Response{Error: "not allowed"})
		_ = conn.Close()
		return
	}

	var req Request
	if err := json.NewDecoder(conn).Decode(&req); err != nil {
		_ = conn.Close()
		return
	}
	switch req.Op {
	case OpRun:
		s.run(conn, req.Config)
		return
	case OpStatus:
		s.mu.Lock()
		running := s.running != nil
		s.mu.Unlock()
		_ = send(conn, Response{OK: true, Version: Version, Running: running})
	case OpStage:
		path, err := s.stage(req.Config)
		_ = send(conn, result(err, Response{Path: path}))
	default:
		_ = send(conn, Response{Error: "unknown op " + req.Op})
	}
	_ = conn.Close()
}

func (s *Server) configPath() string { return filepath.Join(s.WorkDir, "config.yaml") }

func (s *Server) stage(cfg []byte) (string, error) {
	s.mu.Lock()
	running := s.running != nil
	s.mu.Unlock()
	if !running {
		return "", errors.New("core is not running")
	}
	if err := fsutil.WriteFileAtomic(s.configPath(), cfg, 0o600); err != nil {
		return "", err
	}
	return s.configPath(), nil
}

func (s *Server) run(conn net.Conn, cfg []byte) {
	defer conn.Close()
	sess, err := s.start(cfg)
	if err != nil {
		_ = send(conn, Response{Error: err.Error()})
		return
	}
	if err := send(conn, Response{OK: true, Path: s.configPath()}); err != nil {
		sess.stop()
		return
	}

	requests := make(chan Request)
	go func() {
		defer close(requests)
		dec := json.NewDecoder(conn)
		for {
			var req Request
			if dec.Decode(&req) != nil {
				return
			}
			requests <- req
		}
	}()

	for {
		select {
		case <-sess.exited:
			s.release(sess)
			_ = send(conn, Response{Error: "core exited unexpectedly", Tail: sess.logs.Tail(errorTail)})
			return
		case req, ok := <-requests:
			if ok && req.Op != OpStop {
				continue
			}
			sess.stop()
			s.release(sess)
			if ok {
				_ = send(conn, Response{OK: true})
			}
			return
		}
	}
}

func (s *Server) start(cfg []byte) (*session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running != nil {
		return nil, errors.New("core is already running")
	}

	if err := os.MkdirAll(s.WorkDir, 0o700); err != nil {
		return nil, fmt.Errorf("create work dir: %w", err)
	}
	if err := fsutil.WriteFileAtomic(s.configPath(), cfg, 0o600); err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if out, err := exec.CommandContext(ctx, s.Core, "-t", "-d", s.WorkDir, "-f", s.configPath()).CombinedOutput(); err != nil {
		return nil, fmt.Errorf("config rejected: %s", lastLines(string(out)))
	}

	logs := core.NewLogBuffer(logLimit)
	// The core outlives this call; the session owns its lifetime.
	cmd := exec.CommandContext(context.WithoutCancel(ctx), s.Core, "-d", s.WorkDir, "-f", s.configPath())
	cmd.Stdout, cmd.Stderr = logs, logs
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start core: %w", err)
	}
	sess := &session{cmd: cmd, exited: make(chan struct{}), logs: logs}
	go func() {
		_ = cmd.Wait()
		close(sess.exited)
	}()
	s.running = sess
	return sess, nil
}

func (s *Server) release(sess *session) {
	s.mu.Lock()
	if s.running == sess {
		s.running = nil
	}
	s.mu.Unlock()
}

func (sess *session) stop() {
	sess.stopOnce.Do(func() {
		_ = sess.cmd.Process.Signal(os.Interrupt)
		select {
		case <-sess.exited:
		case <-time.After(stopTimeout):
			_ = sess.cmd.Process.Kill()
			<-sess.exited
		}
	})
}

func result(err error, ok Response) Response {
	if err != nil {
		return Response{Error: err.Error()}
	}
	ok.OK = true
	return ok
}

func lastLines(text string) string {
	lines := strings.Split(strings.TrimSpace(text), "\n")
	return strings.Join(lines[max(0, len(lines)-errorTail):], "\n")
}
