package main

import (
	"bufio"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"timo/internal/timo"
)

func TestReadLinesKeepsOrderAndReportsEOF(t *testing.T) {
	lines := readLines(bufio.NewScanner(strings.NewReader("start\npause\n")))
	for _, expected := range []string{"start", "pause"} {
		input, ok := <-lines
		if !ok || input.err != nil || input.text != expected {
			t.Fatalf("got %#v, open=%t; want %q", input, ok, expected)
		}
	}
	if _, ok := <-lines; ok {
		t.Fatal("input channel should close at EOF")
	}
}

func TestWatchPreservesTypedCommands(t *testing.T) {
	store := timo.NewStore(filepath.Join(t.TempDir(), "data.json"))
	_, err := store.Update(time.Now(), func(s *timo.State) error {
		return s.Action("start", time.Now())
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := make(chan inputLine, 1)
	lines <- inputLine{text: "pause"}
	term := newTerminal(store)
	pending, eof, err := term.watch(lines)
	if err != nil || eof || pending != "pause" {
		t.Fatalf("watch: pending=%q eof=%t err=%v", pending, eof, err)
	}
	v, err := store.Snapshot(time.Now())
	if err != nil || v.Timer.Status != timo.StatusRunning {
		t.Fatalf("watch must not change the timer: state=%#v err=%v", v.Timer, err)
	}
}

func TestWatchDoesNotWaitForInputAfterEOF(t *testing.T) {
	store := timo.NewStore(filepath.Join(t.TempDir(), "data.json"))
	_, err := store.Update(time.Now(), func(s *timo.State) error {
		return s.Action("start", time.Now())
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := make(chan inputLine)
	close(lines)
	pending, eof, err := newTerminal(store).watch(lines)
	if err != nil || !eof || pending != "" {
		t.Fatalf("watch: pending=%q eof=%t err=%v", pending, eof, err)
	}
}
