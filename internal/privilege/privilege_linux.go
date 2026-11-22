//go:build linux

package privilege

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"ctun/internal/helper"
)

func Check(context.Context) State {
	info, err := os.Stat(helper.LinuxCore)
	if err != nil || info.Mode()&0o111 == 0 {
		return StateMissing
	}
	return StateInstalled
}

func Install(ctx context.Context, helperBinary, coreBinary string) error {
	return pkexec(ctx, helperBinary, "install", "--core", coreBinary)
}

func Uninstall(ctx context.Context) error {
	helperBinary, err := FindHelper()
	if err != nil {
		return err
	}
	return pkexec(ctx, helperBinary, "uninstall")
}

func pkexec(ctx context.Context, args ...string) error {
	out, err := exec.CommandContext(ctx, "pkexec", args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("privilege: %s", strings.TrimSpace(string(out)))
	}
	return nil
}
