package core

import (
	"bytes"
	"slices"
	"sync"
)

type LogBuffer struct {
	mu      sync.Mutex
	lines   []string
	limit   int
	partial []byte
}

func NewLogBuffer(limit int) *LogBuffer {
	return &LogBuffer{limit: limit}
}

func (b *LogBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	data := slices.Concat(b.partial, p)
	for {
		i := bytes.IndexByte(data, '\n')
		if i < 0 {
			break
		}
		if line := bytes.TrimRight(data[:i], "\r"); len(line) > 0 {
			b.lines = append(b.lines, string(line))
		}
		data = data[i+1:]
	}
	b.partial = append(b.partial[:0], data...)
	if extra := len(b.lines) - b.limit; extra > 0 {
		b.lines = append(b.lines[:0:0], b.lines[extra:]...)
	}
	return len(p), nil
}

func (b *LogBuffer) Tail(n int) []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	start := max(0, len(b.lines)-n)
	return append([]string(nil), b.lines[start:]...)
}
