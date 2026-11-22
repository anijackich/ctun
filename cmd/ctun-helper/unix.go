//go:build darwin || linux

package main

import (
	"flag"
	"fmt"
	"io"
	"os"
)

type options struct {
	uid  uint
	core string
}

func parse(name string, args []string) options {
	var o options
	fs := flag.NewFlagSet(name, flag.ExitOnError)
	fs.UintVar(&o.uid, "uid", 0, "the only user allowed to control the core")
	fs.StringVar(&o.core, "core", "", "mihomo binary to install")
	_ = fs.Parse(args)
	return o
}

func requireRoot() error {
	if os.Geteuid() != 0 {
		return fmt.Errorf("must run as root")
	}
	return nil
}

func installFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open %s: %w", src, err)
	}
	defer in.Close()
	tmp := dst + ".new"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, perm)
	if err != nil {
		return fmt.Errorf("create %s: %w", tmp, err)
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return fmt.Errorf("copy %s: %w", src, err)
	}
	if err := out.Close(); err != nil {
		return fmt.Errorf("close %s: %w", tmp, err)
	}
	if err := os.Chown(tmp, 0, 0); err != nil {
		return fmt.Errorf("chown %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, dst); err != nil {
		return fmt.Errorf("install %s: %w", dst, err)
	}
	return nil
}
