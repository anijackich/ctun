package core

import (
	"slices"
	"testing"
)

func TestLogBufferJoinsPartialWritesAndKeepsTail(t *testing.T) {
	b := NewLogBuffer(2)
	for _, chunk := range []string{"one\ntw", "o\r\n", "\nthree\nfou"} {
		if _, err := b.Write([]byte(chunk)); err != nil {
			t.Fatal(err)
		}
	}
	if got := b.Tail(10); !slices.Equal(got, []string{"two", "three"}) {
		t.Fatalf("Tail = %q", got)
	}
	if _, err := b.Write([]byte("r\n")); err != nil {
		t.Fatal(err)
	}
	if got := b.Tail(1); !slices.Equal(got, []string{"four"}) {
		t.Fatalf("Tail(1) = %q", got)
	}
}
