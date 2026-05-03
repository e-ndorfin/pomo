package ui

import (
	"testing"
	"time"

	"github.com/Bahaaio/pomo/config"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModelInitializesSessionStartedAt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := testConfig()
	config.C = cfg

	before := time.Now()
	workModel := NewModel(config.WorkTask, cfg)
	after := time.Now()

	if workModel.sessionStartedAt.IsZero() {
		t.Fatal("expected work model sessionStartedAt to be initialized")
	}
	if workModel.sessionStartedAt.Before(before) || workModel.sessionStartedAt.After(after) {
		t.Fatalf("expected work model sessionStartedAt between %v and %v, got %v", before, after, workModel.sessionStartedAt)
	}

	before = time.Now()
	breakModel := NewModel(config.BreakTask, cfg)
	after = time.Now()

	if breakModel.sessionStartedAt.IsZero() {
		t.Fatal("expected break model sessionStartedAt to be initialized")
	}
	if breakModel.sessionStartedAt.Before(before) || breakModel.sessionStartedAt.After(after) {
		t.Fatalf("expected break model sessionStartedAt between %v and %v, got %v", before, after, breakModel.sessionStartedAt)
	}
}

func TestResetUpdatesSessionStartedAt(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	cfg := testConfig()
	config.C = cfg

	m := NewModel(config.WorkTask, cfg)
	originalStartedAt := time.Now().Add(-10 * time.Minute)
	m.sessionStartedAt = originalStartedAt
	m.elapsed = 5 * time.Minute
	m.duration = 20 * time.Minute

	beforeReset := time.Now()
	m.handleKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	afterReset := time.Now()

	if m.elapsed != 0 {
		t.Fatalf("expected reset to clear elapsed, got %v", m.elapsed)
	}
	if m.duration != m.currentTask.Duration {
		t.Fatalf("expected reset to restore duration %v, got %v", m.currentTask.Duration, m.duration)
	}
	if !m.sessionStartedAt.After(originalStartedAt) {
		t.Fatalf("expected reset to advance sessionStartedAt past %v, got %v", originalStartedAt, m.sessionStartedAt)
	}
	if m.sessionStartedAt.Before(beforeReset) || m.sessionStartedAt.After(afterReset) {
		t.Fatalf("expected reset sessionStartedAt between %v and %v, got %v", beforeReset, afterReset, m.sessionStartedAt)
	}
}

func testConfig() config.Config {
	return config.Config{
		OnSessionEnd:  "quit",
		CountIdleTime: true,
		Work: config.Task{
			Title:    "work session",
			Duration: 25 * time.Minute,
		},
		Break: config.Task{
			Title:    "break session",
			Duration: 5 * time.Minute,
		},
		LongBreak: config.LongBreak{
			Enabled:  false,
			After:    4,
			Duration: 15 * time.Minute,
		},
	}
}
