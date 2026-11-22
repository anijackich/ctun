package sysnet

import (
	"context"
	"strings"
)

func networkServices(ctx context.Context, run runner) ([]string, error) {
	out, err := run(ctx, "networksetup", "-listallnetworkservices")
	if err != nil {
		return nil, err
	}
	var services []string
	for i, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if i == 0 || line == "" || strings.HasPrefix(line, "*") {
			continue
		}
		services = append(services, line)
	}
	return services, nil
}

func listLines(out string) []string {
	var values []string
	for line := range strings.Lines(out) {
		line = strings.TrimSpace(line)
		if line != "" && !strings.Contains(line, " ") {
			values = append(values, line)
		}
	}
	return values
}
