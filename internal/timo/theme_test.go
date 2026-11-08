package timo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLegacyJSONAppearanceAndPreservedSessions(t *testing.T) {
	p := filepath.Join(t.TempDir(), "data.json")
	legacy := Defaults()
	legacy.Settings.Theme = ""
	legacy.Settings.GrainMotion = false
	legacy.CompletedFocus = 1
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	legacy.History = append(legacy.History, Session{ID: "abc", Phase: PhaseFocus, DurationSeconds: 1500, StartedAt: now.Add(-25 * time.Minute), CompletedAt: now})
	b, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0600); err != nil {
		t.Fatal(err)
	}
	store := NewStore(p)
	got, err := store.Snapshot(now)
	if err != nil {
		t.Fatal(err)
	}
	if got.CompletedFocus != 1 || len(got.History) != 1 {
		t.Fatal("lost existing history")
	}
	got, err = store.Update(now, func(s *State) error {
		cfg := s.Settings
		cfg.Theme = "light"
		cfg.GrainMotion = true
		return s.SetSettings(cfg)
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Settings.Theme != "light" || !got.Settings.GrainMotion {
		t.Fatal("appearance settings were not saved")
	}
	persisted, err := NewStore(p).Snapshot(now)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Settings.Theme != "light" || persisted.CompletedFocus != 1 || len(persisted.History) != 1 {
		t.Fatal("appearance or history was not persisted")
	}
}

func TestInvalidThemeRejected(t *testing.T) {
	s := Defaults()
	config := s.Settings
	config.Theme = "purple"
	if err := s.SetSettings(config); err == nil {
		t.Fatal("unknown theme was accepted")
	}
	config.Theme = "system"
	if err := s.SetSettings(config); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultPathLegacyFallback(t *testing.T) {
	base := t.TempDir()
	t.Setenv("TIMO_DATA_FILE", "")
	t.Setenv("POMO_DATA_FILE", "")
	t.Setenv("HOME", base)
	t.Setenv("XDG_CONFIG_HOME", base)
	t.Setenv("APPDATA", base)
	path, err := DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	legacy := filepath.Join(filepath.Dir(filepath.Dir(path)), "pomo", "data.json")
	if err := os.MkdirAll(filepath.Dir(legacy), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacy, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err := DefaultPath()
	if err != nil || got != legacy {
		t.Fatalf("legacy fallback: got %q, want %q: %v", got, legacy, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	got, err = DefaultPath()
	if err != nil || got != path {
		t.Fatalf("timo path should take precedence: %q %v", got, err)
	}
	t.Setenv("TIMO_DATA_FILE", filepath.Join(base, "custom.json"))
	got, err = DefaultPath()
	if err != nil || got != filepath.Join(base, "custom.json") {
		t.Fatalf("explicit path precedence: %q %v", got, err)
	}
}
