package app

import (
	"context"
	"log/slog"
	"time"

	"ctun/internal/core"
)

const statsInterval = time.Second

func (s *CoreService) streamTraffic(ctx context.Context, client *core.Client) {
	err := client.StreamTraffic(ctx, func(t core.Traffic) { emit(EventTraffic, TrafficEvent(t)) })
	if err != nil && ctx.Err() == nil {
		slog.Warn("traffic stream ended", "err", err)
	}
}

func (s *CoreService) streamStats(ctx context.Context, client *core.Client) {
	go func() {
		_ = client.StreamMemory(ctx, func(m core.Memory) {
			s.mu.Lock()
			s.memory = m.InUse
			s.mu.Unlock()
		})
	}()

	ticker := time.NewTicker(statsInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
		snap, err := client.Connections(ctx)
		if err != nil {
			continue
		}
		s.mu.Lock()
		memory := s.memory
		s.mu.Unlock()
		emit(EventStats, StatsEvent{Connections: len(snap.Connections), Memory: memory})
	}
}
