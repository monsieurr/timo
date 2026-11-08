package timo

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
)

const (
	PhaseFocus     = "focus"
	PhaseBreak     = "break"
	PhaseLongBreak = "long_break"
	StatusIdle     = "idle"
	StatusRunning  = "running"
	StatusPaused   = "paused"
)

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// State is the single, human-readable data format shared by both interfaces.
type State struct {
	Version          int       `json:"version"`
	Settings         Settings  `json:"settings"`
	SettingsRevision int64     `json:"settings_revision"`
	Timer            Timer     `json:"timer"`
	CompletedFocus   int       `json:"completed_focus"`
	History          []Session `json:"history"`
}

type Settings struct {
	FocusMinutes     int    `json:"focus_minutes"`
	BreakMinutes     int    `json:"break_minutes"`
	LongBreakMinutes int    `json:"long_break_minutes"`
	LongBreakEvery   int    `json:"long_break_every"`
	Accent           string `json:"accent"`
	Theme            string `json:"theme,omitempty"`
	GrainMotion      bool   `json:"grain_motion"`
}

type Timer struct {
	ID               string     `json:"id,omitempty"`
	Phase            string     `json:"phase"`
	Status           string     `json:"status"`
	DurationSeconds  int64      `json:"duration_seconds"`
	RemainingSeconds int64      `json:"remaining_seconds"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	EndsAt           *time.Time `json:"ends_at,omitempty"`
}

type Session struct {
	ID              string    `json:"id"`
	Phase           string    `json:"phase"`
	DurationSeconds int64     `json:"duration_seconds"`
	StartedAt       time.Time `json:"started_at"`
	CompletedAt     time.Time `json:"completed_at"`
}

func Defaults() State {
	s := State{Version: 1, Settings: Settings{FocusMinutes: 25, BreakMinutes: 5, LongBreakMinutes: 15, LongBreakEvery: 4, Accent: "#C0B2E8", Theme: "dark"}, History: []Session{}}
	s.Timer = s.newIdle(PhaseFocus)
	return s
}

func (s *State) minutesFor(phase string) int {
	switch phase {
	case PhaseBreak:
		return s.Settings.BreakMinutes
	case PhaseLongBreak:
		return s.Settings.LongBreakMinutes
	default:
		return s.Settings.FocusMinutes
	}
}
func (s *State) newIdle(phase string) Timer {
	duration := int64(s.minutesFor(phase) * 60)
	return Timer{Phase: phase, Status: StatusIdle, DurationSeconds: duration, RemainingSeconds: duration}
}
func (s *State) nextPhase() string {
	if s.Timer.Phase == PhaseFocus {
		if s.CompletedFocus > 0 && s.CompletedFocus%s.Settings.LongBreakEvery == 0 {
			return PhaseLongBreak
		}
		return PhaseBreak
	}
	return PhaseFocus
}

func Remaining(t Timer, now time.Time) int64 {
	if t.Status != StatusRunning || t.EndsAt == nil {
		return t.RemainingSeconds
	}
	secs := int64(math.Ceil(t.EndsAt.Sub(now).Seconds()))
	if secs < 0 {
		return 0
	}
	if secs > t.DurationSeconds {
		return t.DurationSeconds
	}
	return secs
}

// Settle completes an expired session exactly once within the caller's locked transaction.
func (s *State) Settle(now time.Time) bool {
	if s.Timer.Status != StatusRunning || s.Timer.EndsAt == nil || s.Timer.EndsAt.After(now) {
		return false
	}
	end := *s.Timer.EndsAt
	start := end.Add(-time.Duration(s.Timer.DurationSeconds) * time.Second)
	if s.Timer.StartedAt != nil {
		start = *s.Timer.StartedAt
	}
	s.History = append(s.History, Session{ID: s.Timer.ID, Phase: s.Timer.Phase, DurationSeconds: s.Timer.DurationSeconds, StartedAt: start, CompletedAt: end})
	if s.Timer.Phase == PhaseFocus {
		s.CompletedFocus++
	}
	s.Timer = s.newIdle(s.nextPhase())
	return true
}

func randomID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return hex.EncodeToString(b[:]), nil
}

func (s *State) Action(name string, now time.Time) error {
	switch name {
	case "start":
		if s.Timer.Status != StatusIdle {
			return errors.New("timer is not idle; use resume or reset")
		}
		id, err := randomID()
		if err != nil {
			return err
		}
		start := now.UTC()
		end := start.Add(time.Duration(s.Timer.DurationSeconds) * time.Second)
		s.Timer.ID = id
		s.Timer.Status = StatusRunning
		s.Timer.StartedAt = &start
		s.Timer.EndsAt = &end
		s.Timer.RemainingSeconds = s.Timer.DurationSeconds
	case "pause":
		if s.Timer.Status != StatusRunning {
			return errors.New("timer is not running")
		}
		s.Timer.RemainingSeconds = Remaining(s.Timer, now)
		s.Timer.Status = StatusPaused
		s.Timer.EndsAt = nil
	case "resume":
		if s.Timer.Status != StatusPaused {
			return errors.New("timer is not paused")
		}
		end := now.UTC().Add(time.Duration(s.Timer.RemainingSeconds) * time.Second)
		s.Timer.EndsAt = &end
		s.Timer.Status = StatusRunning
	case "reset":
		s.Timer = s.newIdle(s.Timer.Phase)
	case "stop":
		s.Timer = s.newIdle(PhaseFocus)
	case "skip":
		// A skipped focus earns no credit and must not repeat the prior long break.
		phase := PhaseFocus
		if s.Timer.Phase == PhaseFocus {
			phase = PhaseBreak
		}
		s.Timer = s.newIdle(phase)
	case "focus", "break", "long_break":
		s.Timer = s.newIdle(name)
	default:
		return fmt.Errorf("unknown action %q", name)
	}
	return nil
}

func ValidSettings(v Settings) error {
	if v.FocusMinutes < 1 || v.FocusMinutes > 240 {
		return errors.New("focus must be 1–240 minutes")
	}
	if v.BreakMinutes < 1 || v.BreakMinutes > 120 {
		return errors.New("break must be 1–120 minutes")
	}
	if v.LongBreakMinutes < 1 || v.LongBreakMinutes > 120 {
		return errors.New("long break must be 1–120 minutes")
	}
	if v.LongBreakEvery < 1 || v.LongBreakEvery > 20 {
		return errors.New("long break interval must be 1–20 sessions")
	}
	if v.Theme != "" && v.Theme != "dark" && v.Theme != "light" && v.Theme != "system" {
		return errors.New("theme must be dark, light or system")
	}
	if !colorPattern.MatchString(v.Accent) {
		return errors.New("accent must be a six-digit hex colour, e.g. #79C9B1")
	}
	return nil
}

func (s *State) SetSettings(v Settings) error {
	v.Accent = strings.ToUpper(v.Accent)
	if v.Theme == "" {
		v.Theme = "dark" // Existing v1 JSON files did not include appearance settings.
	}
	if err := ValidSettings(v); err != nil {
		return err
	}
	s.Settings = v
	s.SettingsRevision++
	if s.Timer.Status == StatusIdle {
		s.Timer = s.newIdle(s.Timer.Phase)
	}
	return nil
}

func Validate(s State) error {
	if s.Version != 1 {
		return fmt.Errorf("unsupported JSON version %d", s.Version)
	}
	if err := ValidSettings(s.Settings); err != nil {
		return fmt.Errorf("invalid settings: %w", err)
	}
	switch s.Timer.Phase {
	case PhaseFocus, PhaseBreak, PhaseLongBreak:
	default:
		return errors.New("invalid timer phase")
	}
	switch s.Timer.Status {
	case StatusIdle, StatusPaused, StatusRunning:
	default:
		return errors.New("invalid timer status")
	}
	if s.Timer.DurationSeconds < 1 || s.Timer.DurationSeconds > 240*60 || s.Timer.RemainingSeconds < 0 || s.Timer.RemainingSeconds > s.Timer.DurationSeconds {
		return errors.New("invalid timer duration")
	}
	if s.Timer.Status == StatusRunning && (s.Timer.EndsAt == nil || s.Timer.StartedAt == nil || s.Timer.ID == "") {
		return errors.New("running timer is missing timestamps or id")
	}
	if s.CompletedFocus < 0 || s.CompletedFocus > len(s.History) {
		return errors.New("invalid completed focus count")
	}
	return nil
}
