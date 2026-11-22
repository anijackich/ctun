package app

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"
)

const pidLookupTimeout = 2 * time.Second

type pidCache struct {
	mu   sync.Mutex
	byID map[string]int
	scan func(context.Context) map[string]int
}

func newPIDCache() *pidCache {
	return &pidCache{byID: map[string]int{}, scan: scanLocalPorts}
}

func (c *pidCache) resolve(ports map[string]string) map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	var missing bool
	for id := range ports {
		if _, ok := c.byID[id]; !ok {
			missing = true
			break
		}
	}
	if missing {
		ctx, cancel := context.WithTimeout(context.Background(), pidLookupTimeout)
		owners := c.scan(ctx)
		cancel()
		for id, port := range ports {
			if _, ok := c.byID[id]; !ok {
				c.byID[id] = owners[port]
			}
		}
	}
	for id := range c.byID {
		if _, ok := ports[id]; !ok {
			delete(c.byID, id)
		}
	}
	result := make(map[string]int, len(ports))
	for id := range ports {
		result[id] = c.byID[id]
	}
	return result
}

func scanLocalPorts(ctx context.Context) map[string]int {
	// lsof exits non-zero when some sockets are not visible but still prints the rest.
	out, _ := exec.CommandContext(ctx, "lsof", "-nP", "-iTCP", "-iUDP", "-F", "pn").Output()
	return parseLsof(string(out))
}

func parseLsof(out string) map[string]int {
	owners := map[string]int{}
	pid := 0
	for line := range strings.Lines(out) {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		switch line[0] {
		case 'p':
			pid, _ = strconv.Atoi(line[1:])
		case 'n':
			local, _, _ := strings.Cut(line[1:], "->")
			if i := strings.LastIndexByte(local, ':'); i >= 0 && pid > 0 {
				if _, taken := owners[local[i+1:]]; !taken {
					owners[local[i+1:]] = pid
				}
			}
		}
	}
	return owners
}
