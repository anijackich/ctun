package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	exitInfoURL     = "https://ipinfo.io/json"
	exitInfoTimeout = 10 * time.Second
)

type ExitInfo struct {
	IP      string `json:"ip"`
	Country string `json:"country"`
	City    string `json:"city"`
	Org     string `json:"org"`
}

func (s *CoreService) ExitInfo() (ExitInfo, error) {
	client, creds := s.running()
	if client == nil {
		return ExitInfo{}, errNotRunning
	}
	transport := &http.Transport{}
	if proxy := creds.proxyURL(); proxy != "" {
		u, err := url.Parse(proxy)
		if err != nil {
			return ExitInfo{}, fmt.Errorf("exit info: %w", err)
		}
		transport.Proxy = http.ProxyURL(u)
	}
	defer transport.CloseIdleConnections()

	ctx, cancel := context.WithTimeout(context.Background(), exitInfoTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, exitInfoURL, nil)
	if err != nil {
		return ExitInfo{}, fmt.Errorf("exit info: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		return ExitInfo{}, fmt.Errorf("exit info: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ExitInfo{}, fmt.Errorf("exit info: %s", resp.Status)
	}
	var info ExitInfo
	if err := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&info); err != nil {
		return ExitInfo{}, fmt.Errorf("exit info: %w", err)
	}
	return info, nil
}
