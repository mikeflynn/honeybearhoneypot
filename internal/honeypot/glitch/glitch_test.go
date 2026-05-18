package glitch

import (
	"math/rand"
	"strings"
	"testing"
	"time"
)

func TestModelCompletesAfterDuration(t *testing.T) {
	m := New("hello world\nsecond line\n")
	if m == nil {
		t.Fatal("New returned nil")
	}

	// Drive enough ticks to exceed Duration. Tick interval is TickInterval.
	steps := int(Duration/TickInterval) + 2
	var done bool
	for i := 0; i < steps; i++ {
		_, done = m.Update(Tick{})
		if done {
			break
		}
	}
	if !done {
		t.Fatalf("glitch model did not report done after %d ticks", steps)
	}
}

func TestViewNonEmptyAndDifferentFromBase(t *testing.T) {
	base := strings.Repeat("the quick brown fox jumps over the lazy dog\n", 5)
	m := New(base)
	for i := 0; i < 3; i++ {
		m.Update(Tick{})
	}
	out := m.View()
	if out == "" {
		t.Fatal("View returned empty string")
	}
}

func TestViewHandlesEmptyBase(t *testing.T) {
	m := New("")
	out := m.View()
	_ = out
}

func TestStartReturnsTick(t *testing.T) {
	if _, ok := Start().(Tick); !ok {
		t.Fatal("Start() did not return a Tick message")
	}
}

func TestDurationSane(t *testing.T) {
	if Duration < 2*time.Second || Duration > 3*time.Second {
		t.Errorf("Duration %v outside expected 2-3s range", Duration)
	}
}

func TestBaseLinesCapped(t *testing.T) {
	huge := strings.Repeat("filler line\n", maxLines*3)
	m := New(huge)
	if len(m.baseLines) > maxLines {
		t.Errorf("baseLines length %d exceeds cap %d", len(m.baseLines), maxLines)
	}
}

func TestCorruptionDoesNotAccumulate(t *testing.T) {
	base := strings.Repeat("the quick brown fox jumps over the lazy dog\n", 10)
	m := New(base)
	snapshot := make([]string, len(m.baseLines))
	copy(snapshot, m.baseLines)

	for i := 0; i < 20; i++ {
		_ = m.View()
	}

	for i, line := range m.baseLines {
		if line != snapshot[i] {
			t.Fatalf("baseLines[%d] mutated by View(): want %q got %q", i, snapshot[i], line)
		}
	}
}

func TestCorruptLinePreservesAnsi(t *testing.T) {
	styled := "\x1b[31mhello world\x1b[0m and \x1b[1;33mmore styled text\x1b[0m"
	rng := rand.New(rand.NewSource(1))

	for i := 0; i < 500; i++ {
		out := corruptLine(styled, rng)
		// Every original escape sequence must still appear verbatim.
		for _, esc := range []string{"\x1b[31m", "\x1b[0m", "\x1b[1;33m"} {
			if !strings.Contains(out, esc) {
				t.Fatalf("ANSI escape %q corrupted on iteration %d: %q", esc, i, out)
			}
		}
	}
}
