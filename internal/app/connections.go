package app

import (
	"cmp"
	"context"

	"ctun/internal/core"
)

type ConnectionView struct {
	ID          string   `json:"id"`
	Host        string   `json:"host"`
	Port        string   `json:"port"`
	Network     string   `json:"network"`
	Inbound     string   `json:"inbound"`
	Process     string   `json:"process"`
	ProcessPath string   `json:"processPath"`
	PID         int      `json:"pid"`
	Chain       []string `json:"chain"`
	Rule        string   `json:"rule"`
	RulePayload string   `json:"rulePayload"`
	Upload      int64    `json:"upload"`
	Download    int64    `json:"download"`
	Start       int64    `json:"start"`
}

func toConnectionView(c core.Connection) ConnectionView {
	m := c.Metadata
	chain := make([]string, len(c.Chains))
	for i, name := range c.Chains {
		chain[len(c.Chains)-1-i] = name
	}
	return ConnectionView{
		ID:          c.ID,
		Host:        cmp.Or(m.Host, m.SniffHost, m.DestinationIP),
		Port:        m.DestinationPort,
		Network:     m.Network,
		Inbound:     m.Type,
		Process:     m.Process,
		ProcessPath: m.ProcessPath,
		Chain:       chain,
		Rule:        c.Rule,
		RulePayload: c.RulePayload,
		Upload:      c.Upload,
		Download:    c.Download,
		Start:       c.Start.UnixMilli(),
	}
}

func (s *CoreService) Connections() ([]ConnectionView, error) {
	client, _ := s.running()
	if client == nil {
		return []ConnectionView{}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), apiTimeout)
	defer cancel()
	snap, err := client.Connections(ctx)
	if err != nil {
		return nil, err
	}
	ports := make(map[string]string, len(snap.Connections))
	for _, c := range snap.Connections {
		ports[c.ID] = c.Metadata.SourcePort
	}
	pids := s.pids.resolve(ports)
	views := make([]ConnectionView, len(snap.Connections))
	for i, c := range snap.Connections {
		views[i] = toConnectionView(c)
		views[i].PID = pids[c.ID]
	}
	return views, nil
}

func (s *CoreService) CloseConnection(id string) error {
	return s.withClient(func(ctx context.Context, c *core.Client) error { return c.CloseConnection(ctx, id) })
}

func (s *CoreService) CloseAllConnections() error {
	return s.withClient(func(ctx context.Context, c *core.Client) error { return c.CloseConnections(ctx) })
}
