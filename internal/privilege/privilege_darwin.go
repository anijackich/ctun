//go:build darwin

package privilege

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"ctun/internal/helper"
)

const readyTimeout = 10 * time.Second

func Check(ctx context.Context) State {
	resp, err := helper.Status(ctx, helper.SocketPath)
	switch {
	case err != nil:
		return StateMissing
	case resp.Version != helper.Version:
		return StateOutdated
	default:
		return StateInstalled
	}
}

func Install(ctx context.Context, helperBinary, coreBinary string) error {
	script := fmt.Sprintf(`do shell script (quoted form of %s) & " install --uid %d --core " & (quoted form of %s) with administrator privileges`,
		appleString(helperBinary), os.Getuid(), appleString(coreBinary))
	if err := osascript(ctx, script); err != nil {
		return err
	}
	deadline := time.Now().Add(readyTimeout)
	for Check(ctx) != StateInstalled {
		if time.Now().After(deadline) {
			return errors.New("privilege: helper installed but not answering")
		}
		time.Sleep(200 * time.Millisecond)
	}
	return nil
}

func Uninstall(ctx context.Context) error {
	program := filepath.Join(helper.InstallDir, helper.HelperName)
	script := fmt.Sprintf(`do shell script (quoted form of %s) & " uninstall" with administrator privileges`, appleString(program))
	return osascript(ctx, script)
}

func osascript(ctx context.Context, script string) error {
	out, err := exec.CommandContext(ctx, "/usr/bin/osascript", "-e", script).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if strings.Contains(msg, "-128") {
			return errors.New("privilege: cancelled")
		}
		return fmt.Errorf("privilege: %s", msg)
	}
	return nil
}

func appleString(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}
