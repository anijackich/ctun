//go:build darwin

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"text/template"

	"ctun/internal/helper"
)

var plist = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{.Label}}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.Program}}</string>
		<string>serve</string>
		<string>--uid</string>
		<string>{{.UID}}</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardErrorPath</key>
	<string>/Library/Logs/{{.Label}}.log</string>
</dict>
</plist>
`))

func serveCmd(args []string) error {
	o := parse("serve", args)
	if err := requireRoot(); err != nil {
		return err
	}
	if o.uid == 0 {
		return errors.New("--uid is required")
	}

	_ = os.Remove(helper.SocketPath)
	var lc net.ListenConfig
	l, err := lc.Listen(context.Background(), "unix", helper.SocketPath)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}

	if err := os.Chmod(helper.SocketPath, 0o666); err != nil {
		return fmt.Errorf("socket permissions: %w", err)
	}

	server := &helper.Server{
		Core:      filepath.Join(helper.InstallDir, helper.CoreName),
		WorkDir:   helper.WorkDir,
		Authorize: helper.AllowUID(uint32(o.uid)),
	}
	go func() {
		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
		<-signals
		server.Shutdown()
		_ = l.Close()
	}()
	slog.Info("serving", "socket", helper.SocketPath, "uid", o.uid)
	return server.Serve(l)
}

func installCmd(args []string) error {
	o := parse("install", args)
	if err := requireRoot(); err != nil {
		return err
	}
	if o.uid == 0 || o.core == "" {
		return errors.New("--uid and --core are required")
	}
	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate self: %w", err)
	}

	if err := os.MkdirAll(helper.InstallDir, 0o755); err != nil {
		return fmt.Errorf("create install dir: %w", err)
	}
	program := filepath.Join(helper.InstallDir, helper.HelperName)
	for src, dst := range map[string]string{self: program, o.core: filepath.Join(helper.InstallDir, helper.CoreName)} {
		if err := installFile(src, dst, 0o755); err != nil {
			return err
		}
	}
	if err := os.MkdirAll(helper.WorkDir, 0o700); err != nil {
		return fmt.Errorf("create work dir: %w", err)
	}

	f, err := os.OpenFile(helper.PlistPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("write plist: %w", err)
	}
	err = plist.Execute(f, map[string]any{"Label": helper.Label, "Program": program, "UID": o.uid})
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		return fmt.Errorf("write plist: %w", err)
	}

	_ = launchctl("bootout", "system/"+helper.Label)
	return launchctl("bootstrap", "system", helper.PlistPath)
}

func uninstall() error {
	if err := requireRoot(); err != nil {
		return err
	}
	_ = launchctl("bootout", "system/"+helper.Label)
	var errs []error
	for _, path := range []string{helper.PlistPath, helper.SocketPath} {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			errs = append(errs, err)
		}
	}
	for _, dir := range []string{helper.InstallDir, helper.WorkDir} {
		if err := os.RemoveAll(dir); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func launchctl(args ...string) error {
	out, err := exec.CommandContext(context.Background(), "/bin/launchctl", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("launchctl %v: %w: %s", args, err, out)
	}
	return nil
}
