package core

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

func BinaryVersion(ctx context.Context, binary string) (string, error) {
	out, err := exec.CommandContext(ctx, binary, "-v").Output()
	if err != nil {
		return "", fmt.Errorf("core: version: %w", err)
	}
	if v := parseVersion(string(out)); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("core: unexpected version output %q", strings.TrimSpace(string(out)))
}

func parseVersion(out string) string {
	for field := range strings.FieldsSeq(out) {
		if len(field) > 1 && field[0] == 'v' && field[1] >= '0' && field[1] <= '9' {
			return field
		}
	}
	return ""
}
