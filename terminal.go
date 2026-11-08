package main

import (
	"bufio"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"timo/internal/timo"
)

type terminal struct {
	store     *timo.Store
	color     bool
	webServer *http.Server
}

func newTerminal(store *timo.Store) *terminal {
	fi, err := os.Stdout.Stat()
	return &terminal{store: store, color: os.Getenv("NO_COLOR") == "" && err == nil && (fi.Mode()&os.ModeCharDevice != 0)}
}
func (t *terminal) accent(s string, hex string) string {
	if !t.color {
		return s
	}
	var r, g, b uint8
	_, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return s
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", r, g, b, s)
}
func clock(secs int64) string {
	if secs < 0 {
		secs = 0
	}
	return fmt.Sprintf("%02d:%02d", secs/60, secs%60)
}
func label(phase string) string {
	switch phase {
	case timo.PhaseFocus:
		return "Focus"
	case timo.PhaseBreak:
		return "Short break"
	default:
		return "Long break"
	}
}
func (t *terminal) show(v timo.State) {
	remaining := timo.Remaining(v.Timer, time.Now())
	ratio := float64(v.Timer.DurationSeconds-remaining) / float64(v.Timer.DurationSeconds)
	n := int(ratio*28 + 0.5)
	if n < 0 {
		n = 0
	}
	if n > 28 {
		n = 28
	}
	bar := t.accent(strings.Repeat("█", n), v.Settings.Accent) + strings.Repeat("░", 28-n)
	fmt.Printf("\n%s  %s  ·  %s\n%s  %s\n\n", t.accent("TIMO", v.Settings.Accent), label(v.Timer.Phase), strings.ToUpper(v.Timer.Status), bar, clock(remaining))
}
func themeOrDefault(theme string) string {
	if theme == "" {
		return "dark"
	}
	return theme
}
func (t *terminal) showSettings(v timo.State) {
	fmt.Printf("\n%s\n  focus    %d min\n  break    %d min\n  long     %d min (every %d completed focus sessions)\n  color    %s\n  theme    %s\n  grain    %t\n\n", t.accent("SETTINGS", v.Settings.Accent), v.Settings.FocusMinutes, v.Settings.BreakMinutes, v.Settings.LongBreakMinutes, v.Settings.LongBreakEvery, t.accent(v.Settings.Accent, v.Settings.Accent), themeOrDefault(v.Settings.Theme), v.Settings.GrainMotion)
}
func (t *terminal) showStats(v timo.State, days int) {
	stats := timo.ComputeStats(v, time.Now(), time.Local)
	fmt.Printf("\n%s\n  Today      %d focused min\n  Total      %d focus sessions · %d focused min\n  Breaks     %d completed\n  Streak     %d day(s)\n", t.accent("YOUR STATS", v.Settings.Accent), stats.TodayMinutes, stats.FocusSessions, stats.FocusMinutes, stats.BreakSessions, stats.Streak)
	if days < 1 {
		days = 7
	}
	if days > 365 {
		days = 365
	}
	now := time.Now()
	fmt.Printf("\n  LAST %d DAYS\n", days)
	totals := map[string]int{}
	max := 1
	for _, entry := range v.History {
		if entry.Phase == timo.PhaseFocus {
			key := entry.CompletedAt.In(time.Local).Format("2006-01-02")
			totals[key] += int(entry.DurationSeconds / 60)
		}
	}
	for d := days - 1; d >= 0; d-- {
		key := now.AddDate(0, 0, -d).Format("2006-01-02")
		if totals[key] > max {
			max = totals[key]
		}
	}
	for d := days - 1; d >= 0; d-- {
		date := now.AddDate(0, 0, -d).Format("2006-01-02")
		minutes := totals[date]
		length := minutes * 24 / max
		if minutes > 0 && length == 0 {
			length = 1
		}
		fmt.Printf("  %s  %s%s %4d min\n", date[5:], t.accent(strings.Repeat("█", length), v.Settings.Accent), strings.Repeat(" ", 24-length), minutes)
	}
	fmt.Println()
}

// inputLine is read by one goroutine for the entire REPL lifetime. It avoids
// competing reads from stdin when a timer finishes while the user is typing.
type inputLine struct {
	text string
	err  error
}

func readLines(scanner *bufio.Scanner) <-chan inputLine {
	ch := make(chan inputLine)
	go func() {
		defer close(ch)
		for scanner.Scan() {
			ch <- inputLine{text: scanner.Text()}
		}
		if err := scanner.Err(); err != nil {
			ch <- inputLine{err: err}
		}
	}()
	return ch
}

// watch displays a live countdown until Enter, a typed command, EOF or
// a change of timer status. A non-empty line is returned to the REPL, so
// piped commands and users typing 'pause' aren't silently discarded.
func (t *terminal) watch(lines <-chan inputLine) (pending string, eof bool, err error) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	last := ""
	var activeID string
	for {
		v, err := t.store.Snapshot(time.Now())
		if err != nil {
			fmt.Println()
			return "", false, err
		}
		if v.Timer.Status != timo.StatusRunning {
			fmt.Println()
			if activeID == "" {
				fmt.Println("No running timer.")
			} else if v.Timer.Status == timo.StatusPaused {
				fmt.Println("Timer paused.")
			} else {
				completed := false
				for _, session := range v.History {
					if session.ID == activeID {
						completed = true
						break
					}
				}
				if completed {
					fmt.Print("\aSession complete. The next phase is ready.\n")
				} else {
					fmt.Println("Timer stopped or changed.")
				}
			}
			return "", false, nil
		}
		activeID = v.Timer.ID
		remaining := timo.Remaining(v.Timer, time.Now())
		line := fmt.Sprintf("  %-12s %-8s  %s   [Enter: commands]", label(v.Timer.Phase), strings.ToUpper(v.Timer.Status), clock(remaining))
		if line != last {
			// Padding clears any previous text without requiring Windows 10's
			// optional VT escape-sequence support.
			fmt.Print("\r" + t.accent(fmt.Sprintf("%-76s", line), v.Settings.Accent))
			last = line
		}
		select {
		case input, ok := <-lines:
			fmt.Println()
			if !ok {
				return "", true, nil
			}
			if input.err != nil {
				return "", false, input.err
			}
			return strings.TrimSpace(input.text), false, nil
		case <-ticker.C:
		}
	}
}
func (t *terminal) command(args []string) (bool, error) {
	if len(args) == 0 {
		return false, nil
	}
	cmd := strings.ToLower(args[0])
	now := time.Now()
	switch cmd {
	case "q", "quit", "exit":
		return true, nil
	case "help", "?":
		printHelp()
		return false, nil
	case "status", "timer":
		v, err := t.store.Snapshot(now)
		if err == nil {
			t.show(v)
		}
		return false, err
	case "watch":
		return false, fmt.Errorf("watch is available in interactive mode; run timo to enter the REPL")
	case "where", "data":
		fmt.Println(t.store.Path)
		return false, nil
	case "stats":
		days := 7
		if len(args) > 1 {
			var err error
			days, err = strconv.Atoi(args[1])
			if err != nil {
				return false, fmt.Errorf("expected number of days")
			}
		}
		if days < 1 || days > 365 {
			return false, fmt.Errorf("days must be 1–365")
		}
		v, err := t.store.Snapshot(now)
		if err == nil {
			t.showStats(v, days)
		}
		return false, err
	case "settings":
		v, err := t.store.Snapshot(now)
		if err == nil {
			t.showSettings(v)
		}
		return false, err
	case "set":
		if len(args) != 3 {
			return false, fmt.Errorf("usage: set focus|break|long|every|color|theme|grain VALUE")
		}
		key, value := strings.ToLower(args[1]), args[2]
		v, err := t.store.Update(now, func(s *timo.State) error {
			cfg := s.Settings
			if key == "color" || key == "colour" {
				cfg.Accent = value
			} else if key == "theme" {
				cfg.Theme = value
			} else if key == "grain" {
				switch value {
				case "on", "true":
					cfg.GrainMotion = true
				case "off", "false":
					cfg.GrainMotion = false
				default:
					return fmt.Errorf("grain must be on or off")
				}
			} else {
				n, err := strconv.Atoi(value)
				if err != nil {
					return fmt.Errorf("expected an integer number of minutes")
				}
				switch key {
				case "focus":
					cfg.FocusMinutes = n
				case "break":
					cfg.BreakMinutes = n
				case "long":
					cfg.LongBreakMinutes = n
				case "every":
					cfg.LongBreakEvery = n
				default:
					return fmt.Errorf("unknown setting %q", key)
				}
			}
			return s.SetSettings(cfg)
		})
		if err == nil {
			t.showSettings(v)
		}
		return false, err
	case "start", "pause", "resume", "reset", "stop", "skip", "focus", "break", "long_break":
		v, err := t.store.Update(now, func(s *timo.State) error { return s.Action(cmd, now) })
		if err == nil {
			t.show(v)
		}
		return false, err
	case "web":
		return false, t.openWeb()
	default:
		return false, fmt.Errorf("unknown command %q (try help)", cmd)
	}
}
func printHelp() {
	fmt.Print(`
COMMANDS
  start          Start the current session and show a live countdown
  pause/resume   Pause or resume
  reset          Reset current session without recording it
  skip           Move to the next phase without recording it
  stop           Return to a fresh focus session
  focus          Select a focus session
  break          Select a short break
  long_break     Select a long break
  status         Show remaining time and progress
  watch          Live countdown (Enter returns to the prompt)
  stats [days]   Display focus activity (default: 7)
  settings       Display your defaults
  set focus 30   Set default focus time in minutes
  set break 5    Set short break time
  set long 15    Set long break time
  set every 4    Set focus sessions before a long break
  set color #C0B2E8   Set the accent colour
  set theme dark      Set dark, light, or system appearance
  set grain on        Toggle subtle background grain motion
  web            Start/open the browser dashboard
  where          Show the JSON data location
  help           Print commands
  quit           Leave the terminal (timer state is preserved)
`)
}
func runREPL(store *timo.Store) error {
	t := newTerminal(store)
	defer t.closeWeb()
	v, err := store.Snapshot(time.Now())
	if err != nil {
		return err
	}
	fmt.Printf("%s  ·  simple focus timer\n", t.accent("TIMO", v.Settings.Accent))
	fmt.Println("Type help for commands, or start to begin.")
	t.show(v)

	lines := readLines(bufio.NewScanner(os.Stdin))
	pending := ""
	if v.Timer.Status == timo.StatusRunning {
		var eof bool
		pending, eof, err = t.watch(lines)
		if err != nil {
			return err
		}
		if eof {
			return nil
		}
	}
	for {
		var text string
		if pending != "" {
			text, pending = pending, ""
		} else {
			fmt.Print("timo > ")
			input, ok := <-lines
			if !ok {
				fmt.Println()
				return nil
			}
			if input.err != nil {
				return input.err
			}
			text = input.text
		}
		parts := strings.Fields(strings.TrimSpace(text))
		if len(parts) == 0 {
			continue
		}
		command := strings.ToLower(parts[0])
		if command == "watch" {
			pending, _, err = t.watch(lines)
		} else {
			var quit bool
			quit, err = t.command(parts)
			if quit {
				return nil
			}
			// A running session should visibly count down immediately, without
			// changing the behaviour of one-shot shell commands (`timo start`).
			if err == nil && (command == "start" || command == "resume") {
				var eof bool
				pending, eof, err = t.watch(lines)
				if eof {
					return nil
				}
			}
		}
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		}
	}
}

// Used by direct subcommands and tests to avoid needing a terminal.
func runCommand(store *timo.Store, parts []string) error {
	t := newTerminal(store)
	_, err := t.command(parts)
	return err
}
