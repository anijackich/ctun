//go:build linux

package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"ctun/internal/helper"
)

func serveCmd([]string) error {
	return errors.New("on Linux the core runs as the user with capabilities; no daemon needed")
}

func installCmd(args []string) error {
	o := parse("install", args)
	if err := requireRoot(); err != nil {
		return err
	}
	if o.core == "" {
		return errors.New("--core is required")
	}
	if err := os.MkdirAll(filepath.Dir(helper.LinuxCore), 0o755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	if err := installFile(o.core, helper.LinuxCore, 0o755); err != nil {
		return err
	}
	out2, err := exec.CommandContext(context.Background(), "setcap", "cap_net_admin,cap_net_bind_service,cap_net_raw+ep", helper.LinuxCore).CombinedOutput()
	if err != nil {
		return fmt.Errorf("setcap: %w: %s", err, out2)
	}
	return nil
}

func uninstall() error {
	if err := requireRoot(); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Dir(helper.LinuxCore)); err != nil {
		return fmt.Errorf("remove core: %w", err)
	}
	return nil
}
