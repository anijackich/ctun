package core

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

type Traffic struct {
	Up        int64 `json:"up"`
	Down      int64 `json:"down"`
	UpTotal   int64 `json:"upTotal"`
	DownTotal int64 `json:"downTotal"`
}

type Memory struct {
	InUse uint64 `json:"inuse"`
}

type ConnectionsSnapshot struct {
	Connections []Connection `json:"connections"`
}

type Connection struct {
	ID       string `json:"id"`
	Metadata struct {
		Network         string `json:"network"`
		Type            string `json:"type"`
		SourcePort      string `json:"sourcePort"`
		DestinationIP   string `json:"destinationIP"`
		DestinationPort string `json:"destinationPort"`
		Host            string `json:"host"`
		SniffHost       string `json:"sniffHost"`
		Process         string `json:"process"`
		ProcessPath     string `json:"processPath"`
	} `json:"metadata"`
	Upload      int64     `json:"upload"`
	Download    int64     `json:"download"`
	Start       time.Time `json:"start"`
	Chains      []string  `json:"chains"`
	Rule        string    `json:"rule"`
	RulePayload string    `json:"rulePayload"`
}

type Rule struct {
	Index   int    `json:"index"`
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Proxy   string `json:"proxy"`
	Extra   *struct {
		Disabled bool      `json:"disabled"`
		HitCount uint64    `json:"hitCount"`
		HitAt    time.Time `json:"hitAt"`
	} `json:"extra"`
}

type RuleProvider struct {
	Name        string    `json:"name"`
	Behavior    string    `json:"behavior"`
	Format      string    `json:"format"`
	VehicleType string    `json:"vehicleType"`
	RuleCount   int       `json:"ruleCount"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type LogEntry struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

type ProxyState struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Now     string   `json:"now"`
	All     []string `json:"all"`
	History []struct {
		Delay int `json:"delay"`
	} `json:"history"`
}

func (p ProxyState) LastDelay() (int, bool) {
	if len(p.History) == 0 {
		return 0, false
	}
	return p.History[len(p.History)-1].Delay, true
}

type Client struct {
	base   string
	secret string
	http   *http.Client
}

func NewClient(addr, secret string) *Client {
	// The controller is local: never route it through a system proxy.
	transport := &http.Transport{Proxy: nil}
	return &Client{base: "http://" + addr, secret: secret, http: &http.Client{Transport: transport}}
}

func (c *Client) Version(ctx context.Context) (string, error) {
	var v struct {
		Version string `json:"version"`
	}
	if err := c.do(ctx, http.MethodGet, "/version", nil, &v); err != nil {
		return "", err
	}
	return v.Version, nil
}

func (c *Client) SetMode(ctx context.Context, mode string) error {
	return c.do(ctx, http.MethodPatch, "/configs", map[string]string{"mode": mode}, nil)
}

func (c *Client) Proxies(ctx context.Context) (map[string]ProxyState, error) {
	var resp struct {
		Proxies map[string]ProxyState `json:"proxies"`
	}
	err := c.do(ctx, http.MethodGet, "/proxies", nil, &resp)
	return resp.Proxies, err
}

func (c *Client) SelectProxy(ctx context.Context, group, name string) error {
	return c.do(ctx, http.MethodPut, "/proxies/"+url.PathEscape(group), map[string]string{"name": name}, nil)
}

func (c *Client) ProxyDelay(ctx context.Context, name, testURL string, timeout time.Duration) (int, error) {
	query := url.Values{"url": {testURL}, "timeout": {strconv.Itoa(int(timeout.Milliseconds()))}}
	var resp struct {
		Delay int `json:"delay"`
	}
	err := c.do(ctx, http.MethodGet, "/proxies/"+url.PathEscape(name)+"/delay?"+query.Encode(), nil, &resp)
	return resp.Delay, err
}

func (c *Client) Reload(ctx context.Context, path string) error {
	return c.do(ctx, http.MethodPut, "/configs?force=true", map[string]string{"path": path}, nil)
}

// CloseConnections drops every open connection, so apps reconnect through the current
// selection instead of keeping alive ones on the previous server.
func (c *Client) CloseConnections(ctx context.Context) error {
	return c.do(ctx, http.MethodDelete, "/connections", nil, nil)
}

func (c *Client) CloseConnection(ctx context.Context, id string) error {
	return c.do(ctx, http.MethodDelete, "/connections/"+url.PathEscape(id), nil, nil)
}

func (c *Client) Rules(ctx context.Context) ([]Rule, error) {
	var out struct {
		Rules []Rule `json:"rules"`
	}
	err := c.do(ctx, http.MethodGet, "/rules", nil, &out)
	return out.Rules, err
}

func (c *Client) SetRuleDisabled(ctx context.Context, index int, disabled bool) error {
	return c.do(ctx, http.MethodPatch, "/rules/disable", map[int]bool{index: disabled}, nil)
}

func (c *Client) RuleProviders(ctx context.Context) (map[string]RuleProvider, error) {
	var out struct {
		Providers map[string]RuleProvider `json:"providers"`
	}
	err := c.do(ctx, http.MethodGet, "/providers/rules", nil, &out)
	return out.Providers, err
}

func (c *Client) UpdateRuleProvider(ctx context.Context, name string) error {
	return c.do(ctx, http.MethodPut, "/providers/rules/"+url.PathEscape(name), nil, nil)
}

func (c *Client) Connections(ctx context.Context) (ConnectionsSnapshot, error) {
	var snap ConnectionsSnapshot
	err := c.do(ctx, http.MethodGet, "/connections", nil, &snap)
	return snap, err
}

func (c *Client) StreamTraffic(ctx context.Context, fn func(Traffic)) error {
	return stream(ctx, c, "/traffic", fn)
}

func (c *Client) StreamLogs(ctx context.Context, level string, fn func(LogEntry)) error {
	return stream(ctx, c, "/logs?level="+url.QueryEscape(level), fn)
}

func (c *Client) StreamMemory(ctx context.Context, fn func(Memory)) error {
	return stream(ctx, c, "/memory", fn)
}

func stream[T any](ctx context.Context, c *Client, path string, fn func(T)) error {
	resp, err := c.send(ctx, http.MethodGet, path, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	dec := json.NewDecoder(resp.Body)
	for {
		var sample T
		if err := dec.Decode(&sample); err != nil {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return fmt.Errorf("core: %s: %w", path, ctxErr)
			}
			if errors.Is(err, io.EOF) {
				return nil
			}
			return fmt.Errorf("core: read %s: %w", path, err)
		}
		fn(sample)
	}
}

func (c *Client) do(ctx context.Context, method, path string, body, out any) error {
	resp, err := c.send(ctx, method, path, body)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("core: decode %s: %w", path, err)
	}
	return nil
}

func (c *Client) send(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("core: encode %s: %w", path, err)
		}
		reader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return nil, fmt.Errorf("core: request %s: %w", path, err)
	}
	req.Header.Set("Authorization", "Bearer "+c.secret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("core: %s %s: %w", method, path, err)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		msg, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("core: %s %s: %s: %s", method, path, resp.Status, bytes.TrimSpace(msg))
	}
	return resp, nil
}
