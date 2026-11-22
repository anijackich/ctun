package app

import (
	"github.com/wailsapp/wails/v3/pkg/application"

	"ctun/internal/core"
	"ctun/internal/settings"
)

const (
	EventStatus   = "core:status"
	EventTraffic  = "core:traffic"
	EventStats    = "core:stats"
	EventLogs     = "core:logs"
	EventProfiles = "profiles:changed"
	EventLatency  = "latency:result"
	EventSettings = "settings:changed"
	EventRouting  = "routing:changed"
)

type StatusEvent struct {
	Status    core.Status `json:"status"`
	StartedAt int64       `json:"startedAt,omitempty"`
	Error     string      `json:"error,omitempty"`
	ErrorCode string      `json:"errorCode,omitempty"`
}

type TrafficEvent struct {
	Up        int64 `json:"up"`
	Down      int64 `json:"down"`
	UpTotal   int64 `json:"upTotal"`
	DownTotal int64 `json:"downTotal"`
}

type StatsEvent struct {
	Connections int    `json:"connections"`
	Memory      uint64 `json:"memory"`
}

type LatencyEvent struct {
	ProfileID string `json:"profileId"`
	Name      string `json:"name"`
	Delay     int    `json:"delay"`
}

func init() {
	application.RegisterEvent[StatusEvent](EventStatus)
	application.RegisterEvent[TrafficEvent](EventTraffic)
	application.RegisterEvent[StatsEvent](EventStats)
	application.RegisterEvent[[]LogView](EventLogs)
	application.RegisterEvent[[]ProfileView](EventProfiles)
	application.RegisterEvent[LatencyEvent](EventLatency)
	application.RegisterEvent[settings.Settings](EventSettings)
	application.RegisterEvent[[]RoutingProfileView](EventRouting)
}

func emit(name string, data any) {
	if app := application.Get(); app != nil {
		app.Event.Emit(name, data)
	}
}
