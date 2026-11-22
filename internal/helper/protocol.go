package helper

import (
	"encoding/json"
	"fmt"
	"net"
)

const Version = "1"

const (
	Label      = "dev.ctun.helper"
	SocketPath = "/var/run/" + Label + ".sock"
	InstallDir = "/Library/PrivilegedHelperTools/" + Label
	WorkDir    = "/Library/Application Support/" + Label
	PlistPath  = "/Library/LaunchDaemons/" + Label + ".plist"
	HelperName = "ctun-helper"
	CoreName   = "mihomo"

	// LinuxCore is the capability-enabled, root-owned copy of the core used for TUN on Linux;
	// rewriting a file drops its capabilities, so the user cannot swap it.
	LinuxCore = "/usr/local/lib/ctun/mihomo"
)

const (
	OpStatus = "status"

	OpRun = "run"

	OpStage = "stage"

	OpStop = "stop"
)

type Request struct {
	Op     string `json:"op"`
	Config []byte `json:"config,omitempty"`
}

type Response struct {
	OK      bool     `json:"ok"`
	Error   string   `json:"error,omitempty"`
	Version string   `json:"version,omitempty"`
	Running bool     `json:"running,omitempty"`
	Path    string   `json:"path,omitempty"`
	Tail    []string `json:"tail,omitempty"`
}

func send(conn net.Conn, v any) error {
	if err := json.NewEncoder(conn).Encode(v); err != nil {
		return fmt.Errorf("helper: send: %w", err)
	}
	return nil
}
