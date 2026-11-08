package timo

import (
	"testing"
	"time"
)

func TestDefaultSessionsAndTransitions(t *testing.T) {
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, time.UTC)
	s := Defaults()
	if s.Settings.FocusMinutes != 25 || s.Settings.BreakMinutes != 5 || s.Timer.Status != StatusIdle {
		t.Fatalf("defaults: %+v", s)
	}
	if err := s.Action("start", now); err != nil {
		t.Fatal(err)
	}
	if got := Remaining(s.Timer, now.Add(61*time.Second)); got != 1439 {
		t.Fatalf("remaining: %d", got)
	}
	if err := s.Action("pause", now.Add(61*time.Second)); err != nil {
		t.Fatal(err)
	}
	if s.Timer.RemainingSeconds != 1439 || s.Settle(now.Add(2*time.Hour)) {
		t.Fatalf("paused timer completed unexpectedly: %+v", s.Timer)
	}
	if err := s.Action("resume", now.Add(3*time.Hour)); err != nil {
		t.Fatal(err)
	}
	end := *s.Timer.EndsAt
	if s.Settle(end.Add(-time.Nanosecond)) {
		t.Fatal("completed early")
	}
	if !s.Settle(end) {
		t.Fatal("expected completion")
	}
	if s.Settle(end.Add(time.Second)) {
		t.Fatal("duplicate completion")
	}
	if len(s.History) != 1 || s.CompletedFocus != 1 || s.Timer.Phase != PhaseBreak || s.Timer.Status != StatusIdle {
		t.Fatalf("invalid transition: %+v", s)
	}
	if err := s.Action("start", end); err != nil {
		t.Fatal(err)
	}
	if !s.Settle(end.Add(5 * time.Minute)) {
		t.Fatal("break not recorded")
	}
	if s.Timer.Phase != PhaseFocus || len(s.History) != 2 {
		t.Fatalf("break transition: %+v", s)
	}
}
func TestLongBreakAtFourthCompletion(t *testing.T) {
	s := Defaults()
	now := time.Date(2026, 11, 7, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 4; i++ {
		if err := s.Action("start", now); err != nil {
			t.Fatal(err)
		}
		now = now.Add(25 * time.Minute)
		if !s.Settle(now) {
			t.Fatal("focus did not complete")
		}
		want := PhaseBreak
		if i == 3 {
			want = PhaseLongBreak
		}
		if s.Timer.Phase != want {
			t.Fatalf("focus %d: got %s want %s", i+1, s.Timer.Phase, want)
		}
		if err := s.Action("start", now); err != nil {
			t.Fatal(err)
		}
		now = now.Add(time.Duration(s.Timer.DurationSeconds) * time.Second)
		if !s.Settle(now) || s.Timer.Phase != PhaseFocus {
			t.Fatal("break transition failed")
		}
	}
	if s.CompletedFocus != 4 || len(s.History) != 8 {
		t.Fatalf("history count wrong: %+v", s)
	}
	if err := s.Action("skip", now); err != nil {
		t.Fatal(err)
	}
	if s.Timer.Phase != PhaseBreak {
		t.Fatalf("skipping focus after fourth completion must not create another long break: %+v", s.Timer)
	}
}
func TestSettingsAndReset(t *testing.T) {
	now := time.Now()
	s := Defaults()
	bad := s.Settings
	bad.Accent = "red"
	if err := s.SetSettings(bad); err == nil {
		t.Fatal("accepted invalid colour")
	}
	if s.SettingsRevision != 0 {
		t.Fatal("invalid settings changed state")
	}
	cfg := s.Settings
	cfg.FocusMinutes = 42
	cfg.Accent = "#aabbcc"
	if err := s.SetSettings(cfg); err != nil {
		t.Fatal(err)
	}
	if s.Settings.Accent != "#AABBCC" || s.Timer.DurationSeconds != 2520 || s.SettingsRevision != 1 {
		t.Fatal("settings not applied")
	}
	if err := s.Action("start", now); err != nil {
		t.Fatal(err)
	}
	cfg.FocusMinutes = 7
	if err := s.SetSettings(cfg); err != nil {
		t.Fatal(err)
	}
	if s.Timer.DurationSeconds != 2520 {
		t.Fatal("settings interrupted running timer")
	}
	if err := s.Action("reset", now); err != nil {
		t.Fatal(err)
	}
	if s.Timer.DurationSeconds != 420 || s.Timer.Status != StatusIdle {
		t.Fatal("reset does not respect updated defaults")
	}
	if len(s.History) != 0 {
		t.Fatal("reset accidentally recorded session")
	}
}
func TestStatsTodayAndStreak(t *testing.T) {
	loc := time.FixedZone("CET", 3600)
	now := time.Date(2026, 11, 7, 12, 0, 0, 0, loc)
	s := Defaults()
	s.History = []Session{
		{ID: "1", Phase: PhaseFocus, DurationSeconds: 1500, CompletedAt: now.AddDate(0, 0, -1)},
		{ID: "2", Phase: PhaseFocus, DurationSeconds: 1500, CompletedAt: now},
		{ID: "3", Phase: PhaseBreak, DurationSeconds: 300, CompletedAt: now},
	}
	got := ComputeStats(s, now, loc)
	if got.FocusSessions != 2 || got.TodayMinutes != 25 || got.FocusMinutes != 50 || got.Streak != 2 || got.BreakSessions != 1 {
		t.Fatalf("stats: %+v", got)
	}
}
