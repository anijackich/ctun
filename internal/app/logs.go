package app

import (
	"context"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"time"

	"ctun/internal/core"
)

const (
	logCapacity   = 2000
	logFlushEvery = 200 * time.Millisecond

	logStreamLevel = "debug"

	sourceCore = "core"
	sourceApp  = "app"
)

type LogView struct {
	Time    int64  `json:"time"`
	Source  string `json:"source"`
	Level   string `json:"level"`
	Message string `json:"message"`
}

type logBook struct {
	mu      sync.Mutex
	entries []LogView
	pending []LogView
}

func (b *logBook) add(at time.Time, source, level, message string) {
	entry := LogView{Time: at.UnixMilli(), Source: source, Level: level, Message: message}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries = keepLast(append(b.entries, entry), logCapacity)
	b.pending = keepLast(append(b.pending, entry), logCapacity)
}

func keepLast(entries []LogView, n int) []LogView {
	if len(entries) > n {
		return entries[len(entries)-n:]
	}
	return entries
}

func (b *logBook) drain() []LogView {
	b.mu.Lock()
	defer b.mu.Unlock()
	batch := b.pending
	b.pending = nil
	return batch
}

func (b *logBook) all() []LogView {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]LogView(nil), b.entries...)
}

func (b *logBook) clear() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries, b.pending = nil, nil
}

func (s *CoreService) Logs() []LogView {
	return s.logs.all()
}

func (s *CoreService) ClearLogs() {
	s.logs.clear()
}

func (s *CoreService) streamLogs(ctx context.Context, client *core.Client) {
	err := client.StreamLogs(ctx, logStreamLevel, func(e core.LogEntry) {
		s.logs.add(time.Now(), sourceCore, e.Type, e.Payload)
	})
	if err != nil && ctx.Err() == nil {
		slog.Debug("log stream ended", "err", err)
	}
}

func (s *CoreService) flushLogs(ctx context.Context) {
	ticker := time.NewTicker(logFlushEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if batch := s.logs.drain(); len(batch) > 0 {
				emit(EventLogs, batch)
			}
		}
	}
}

// LogHandler mirrors the app's own records (info and above) into the core's Logs page. It is a
// function, not a method: Wails would expose a method to the frontend.
func LogHandler(s *CoreService) slog.Handler {
	return &bookHandler{book: &s.logs}
}

type bookHandler struct {
	book   *logBook
	attrs  []slog.Attr
	prefix string
}

func (h *bookHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= slog.LevelInfo
}

func (h *bookHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Message)
	write := func(key string, value slog.Value) {
		b.WriteString(" " + key + "=" + value.String())
	}

	for _, a := range h.attrs {
		write(a.Key, a.Value)
	}
	r.Attrs(func(a slog.Attr) bool {
		write(h.prefix+a.Key, a.Value)
		return true
	})
	h.book.add(r.Time, sourceApp, levelName(r.Level), b.String())
	return nil
}

func (h *bookHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	for _, a := range attrs {
		a.Key = h.prefix + a.Key
		next.attrs = append(slices.Clip(next.attrs), a)
	}
	return &next
}

func (h *bookHandler) WithGroup(name string) slog.Handler {
	next := *h
	next.prefix += name + "."
	return &next
}

func levelName(level slog.Level) string {
	switch {
	case level >= slog.LevelError:
		return "error"
	case level >= slog.LevelWarn:
		return "warning"
	case level >= slog.LevelInfo:
		return "info"
	}
	return "debug"
}
