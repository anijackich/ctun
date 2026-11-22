package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"ctun/internal/core"
	"ctun/internal/privilege"
	"ctun/internal/profile"
	"ctun/internal/routing"
	"ctun/internal/settings"
	"ctun/internal/sysnet"
)

const (
	startTimeout = 15 * time.Second
	stopTimeout  = 8 * time.Second
	apiTimeout   = 5 * time.Second
)

type CoreService struct {
	paths        Paths
	settings     *settings.Store
	profiles     *profile.Store
	routes       *routing.Store
	local        runner
	tun          runner
	proxyManager *sysnet.Manager[sysnet.Proxy]
	proxySync    *osSync[proxyTarget]
	dnsSync      *osSync[string]
	logs         logBook
	stopLogs     context.CancelFunc
	pids         *pidCache

	onConnected func()

	// lifecycle serializes Start, restart and reload, which all rewrite the config. Stop is
	// left out so it can interrupt a start that hangs.
	lifecycle sync.Mutex

	mu          sync.Mutex
	active      runner
	client      *core.Client
	creds       runtimeCreds
	stopStreams context.CancelFunc
	state       StatusEvent
	memory      uint64
}

func NewCoreService(paths Paths, store *settings.Store, profiles *profile.Store, routes *routing.Store) *CoreService {
	s := &CoreService{
		paths:        paths,
		settings:     store,
		profiles:     profiles,
		routes:       routes,
		state:        StatusEvent{Status: core.StatusStopped},
		pids:         newPIDCache(),
		proxyManager: sysnet.NewProxy(filepath.Join(paths.Runtime, "sysproxy.json")),
	}
	s.local = newLocalRunner(paths.Runtime, core.FindBinary, s.onCoreEvent)
	s.tun = newTunRunner(paths.Runtime, s.onCoreEvent)
	s.proxySync = newOSSync(s.systemProxyTarget, applySystemProxy(s.proxyManager),
		func(err error) { _ = s.fail(CodeSystemProxy, err) })
	s.dnsSync = newOSSync(s.systemDNSTarget, applySystemDNS(sysnet.NewDNS(filepath.Join(paths.Runtime, "sysdns.json"))),
		func(err error) { slog.Warn("system DNS", "err", err) })
	return s
}

func (s *CoreService) ServiceStartup(context.Context, application.ServiceOptions) error {
	ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()
	switch stopped, err := core.StopOrphan(ctx, s.paths.Runtime); {
	case err != nil:
		slog.Warn("stop orphaned core", "err", err)
	case stopped:
		slog.Info("stopped a core left running by a previous session")
	}
	s.proxySync.start()
	s.dnsSync.start()
	var logCtx context.Context
	logCtx, s.stopLogs = context.WithCancel(context.Background())
	go s.flushLogs(logCtx)
	return nil
}

func (s *CoreService) ServiceShutdown() error {
	if s.stopLogs != nil {
		s.stopLogs()
	}
	return errors.Join(s.Stop(), s.proxySync.shutdown(), s.dnsSync.shutdown())
}

func (s *CoreService) CoreVersion() string {
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
	if client, _ := s.running(); client != nil {
		v, err := client.Version(ctx)
		if err == nil && v != "" {
			return v
		}
		slog.Warn("read running core version", "err", err)
	}
	binary, err := core.FindBinary()
	if err != nil {
		slog.Warn("locate core", "err", err)
		return ""
	}
	v, err := core.BinaryVersion(ctx, binary)
	if err != nil {
		slog.Warn("read core version", "err", err)
	}
	return v
}

func (s *CoreService) State() StatusEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *CoreService) ProxyURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client == nil {
		return ""
	}
	return s.creds.proxyURL()
}

func (s *CoreService) Start() error {
	s.lifecycle.Lock()
	defer s.lifecycle.Unlock()
	return s.start()
}

func (s *CoreService) start() error {
	opts := s.settings.Get()
	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()

	r := s.local
	if opts.TunnelMode == settings.TunnelTUN {
		if err := s.checkPrivilege(ctx); err != nil {
			return err
		}
		r = s.tun
	} else if _, err := core.FindBinary(); err != nil {
		return s.fail(CodeCoreMissing, err)
	}

	creds, err := newRuntimeCreds(opts)
	if err != nil {
		return s.fail(CodeCoreFailed, err)
	}
	cfg, selected, err := s.buildConfig(opts, creds)
	if err != nil {
		return err
	}

	s.mu.Lock()
	s.active = r
	s.mu.Unlock()
	client, err := r.Start(ctx, cfg, creds.controllerAddr, creds.secret)
	if err != nil {
		return err
	}

	// The core may already have exited, and its Stopped event cleared the client.
	streamCtx, stop := context.WithCancel(context.Background())
	s.mu.Lock()
	if r.Status() != core.StatusRunning {
		s.mu.Unlock()
		stop()
		return s.fail(CodeCoreFailed, errors.New("the core exited right after starting"))
	}
	s.client, s.creds, s.stopStreams = client, creds, stop
	state := s.state
	s.mu.Unlock()
	emit(EventStatus, state)
	s.syncSystemSettings()
	applySelections(client, selected)
	go s.streamTraffic(streamCtx, client)
	go s.streamStats(streamCtx, client)
	go s.streamLogs(streamCtx, client)
	if s.onConnected != nil {
		go s.onConnected()
	}
	return nil
}

func (s *CoreService) Stop() error {
	s.mu.Lock()
	r := s.active
	s.mu.Unlock()
	if r == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()
	return r.Stop(ctx)
}

func (s *CoreService) checkPrivilege(ctx context.Context) error {
	switch privilege.Check(ctx) {
	case privilege.StateInstalled:
		return nil
	case privilege.StateOutdated:
		return s.fail(CodeHelperOutdated, errHelperOutdated)
	case privilege.StateUnsupported:
		return s.fail(CodeTunUnavailable, errTunUnavailable)
	default:
		return s.fail(CodeHelperMissing, errHelperMissing)
	}
}

func (s *CoreService) status() core.Status {
	s.mu.Lock()
	r := s.active
	s.mu.Unlock()
	if r == nil {
		return core.StatusStopped
	}
	return r.Status()
}

func (s *CoreService) restart() error {
	s.lifecycle.Lock()
	defer s.lifecycle.Unlock()
	if s.status() == core.StatusStopped {
		return nil
	}
	if err := s.Stop(); err != nil {
		return err
	}
	return s.start()
}

func (s *CoreService) reload() error {
	s.lifecycle.Lock()
	defer s.lifecycle.Unlock()
	client, creds := s.running()
	if client == nil {
		return nil
	}
	cfg, selected, err := s.buildConfig(s.settings.Get(), creds)
	if err != nil {
		return err
	}
	s.mu.Lock()
	r := s.active
	s.mu.Unlock()
	path, err := r.Stage(cfg)
	if err != nil {
		return fmt.Errorf("stage core config: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), startTimeout)
	defer cancel()
	if err := client.Reload(ctx, path); err != nil {
		return fmt.Errorf("reload core config: %w", err)
	}
	slog.Info("core config reloaded")
	applySelections(client, selected)
	return client.CloseConnections(ctx)
}

func (s *CoreService) applyRoutingMode(mode settings.RoutingMode) error {
	err := s.withClient(func(ctx context.Context, c *core.Client) error { return c.SetMode(ctx, string(mode)) })
	if errors.Is(err, errNotRunning) {
		return nil
	}
	return err
}

func (s *CoreService) running() (*core.Client, runtimeCreds) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client, s.creds
}

func (s *CoreService) withClient(fn func(context.Context, *core.Client) error) error {
	client, _ := s.running()
	if client == nil {
		return errNotRunning
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
	return fn(ctx, client)
}

func (s *CoreService) onCoreEvent(e core.Event) {
	state := StatusEvent{Status: e.Status}
	if e.Err != nil {
		state.Error, state.ErrorCode = e.Err.Error(), CodeCoreFailed
	}
	logCoreEvent(e)

	s.mu.Lock()
	switch e.Status {
	case core.StatusRunning:
		state.StartedAt = time.Now().UnixMilli()
	case core.StatusStopped:
		if s.stopStreams != nil {
			s.stopStreams()
		}
		s.client, s.stopStreams, s.memory, s.creds = nil, nil, 0, runtimeCreds{}
	}
	s.state = state
	s.mu.Unlock()
	emit(EventStatus, state)
	if e.Status == core.StatusStopped {
		s.syncSystemSettings()
	}
}

func (s *CoreService) fail(code string, err error) error {
	state := StatusEvent{Status: s.status(), Error: err.Error(), ErrorCode: code}
	s.mu.Lock()
	if state.Status == core.StatusRunning {
		state.StartedAt = s.state.StartedAt
	}
	s.state = state
	s.mu.Unlock()
	emit(EventStatus, state)
	return err
}

func logCoreEvent(e core.Event) {
	switch {
	case e.Err != nil:
		slog.Error("core stopped with an error", "err", e.Err)
	case e.Status == core.StatusRunning:
		slog.Info("core started")
	case e.Status == core.StatusStopped:
		slog.Info("core stopped")
	}
}
