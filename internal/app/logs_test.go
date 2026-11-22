package app

import (
	"fmt"
	"log/slog"
	"testing"
	"time"

	"ctun/internal/core"
)

func TestLogBookKeepsLatestAndBatches(t *testing.T) {
	var b logBook
	for i := range logCapacity + 10 {
		b.add(time.Now(), sourceCore, "info", fmt.Sprint(i))
	}
	all := b.all()
	if len(all) != logCapacity || all[0].Message != "10" || all[len(all)-1].Message != fmt.Sprint(logCapacity+9) {
		t.Fatalf("kept %d lines from %q to %q", len(all), all[0].Message, all[len(all)-1].Message)
	}
	if batch := b.drain(); len(batch) != logCapacity {
		t.Fatalf("batch = %d lines", len(batch))
	}
	if batch := b.drain(); batch != nil {
		t.Fatalf("second drain = %v", batch)
	}
	b.clear()
	if len(b.all()) != 0 {
		t.Fatal("clear must empty the book")
	}
}

func TestConnectionViewOrdersChainFromGroupToServer(t *testing.T) {
	var c core.Connection
	c.ID, c.Chains, c.Start = "1", []string{"Netherlands 🇳🇱", "PROXY"}, time.UnixMilli(1000)
	c.Metadata.DestinationIP, c.Metadata.DestinationPort = "192.0.2.1", "443"
	v := toConnectionView(c)
	if v.Host != "192.0.2.1" || v.Chain[0] != "PROXY" || v.Chain[1] != "Netherlands 🇳🇱" || v.Start != 1000 {
		t.Fatalf("view = %+v", v)
	}
	c.Metadata.SniffHost = "sniffed.example.com"
	if v := toConnectionView(c); v.Host != "sniffed.example.com" {
		t.Fatalf("sniffed host ignored: %q", v.Host)
	}
	c.Metadata.Host = "example.com"
	if v := toConnectionView(c); v.Host != "example.com" {
		t.Fatalf("host = %q", v.Host)
	}
}

func TestLogHandlerMirrorsAppRecords(t *testing.T) {
	s := &CoreService{}
	log := slog.New(LogHandler(s)).With("subscription", "Test").WithGroup("proxy")
	log.Debug("hidden")
	log.Warn("system proxy failed", "port", 7890)
	lines := s.Logs()
	if len(lines) != 1 {
		t.Fatalf("lines = %+v", lines)
	}
	if l := lines[0]; l.Source != sourceApp || l.Level != "warning" ||
		l.Message != "system proxy failed subscription=Test proxy.port=7890" {
		t.Fatalf("line = %+v", l)
	}
}
