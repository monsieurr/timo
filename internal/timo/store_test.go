package timo

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestStoreSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "timo.json")
	now := time.Date(2026, 11, 7, 14, 0, 0, 0, time.UTC)
	first := NewStore(path)
	_, err := first.Update(now, func(s *State) error { return s.Action("start", now) })
	if err != nil {
		t.Fatal(err)
	}
	reopened := NewStore(path)
	got, err := reopened.Snapshot(now.Add(25 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if got.CompletedFocus != 1 || len(got.History) != 1 || got.Timer.Phase != PhaseBreak {
		t.Fatalf("failed to resume/settle persisted session: %+v", got)
	}
	_, err = reopened.Snapshot(now.Add(25 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var saved State
	if err := json.Unmarshal(b, &saved); err != nil {
		t.Fatal(err)
	}
	if len(saved.History) != 1 {
		t.Fatal("duplicate history across reads")
	}
}
func TestConcurrentWritesAndExactlyOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "timo.json")
	now := time.Date(2026, 11, 7, 14, 0, 0, 0, time.UTC)
	store := NewStore(path)
	const n = 40
	var successes atomic.Int32
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := NewStore(path).Update(now, func(s *State) error { return s.Action("start", now) })
			if err == nil {
				successes.Add(1)
			}
		}()
	}
	wg.Wait()
	if successes.Load() != 1 {
		t.Fatalf("expected one start, got %d", successes.Load())
	}
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := NewStore(path).Snapshot(now.Add(25 * time.Minute))
			if err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	got, err := store.Snapshot(now.Add(25 * time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.History) != 1 || got.CompletedFocus != 1 {
		t.Fatalf("duplicated session: %+v", got)
	}
}
func TestCorruptJSONPreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data.json")
	if err := os.WriteFile(path, []byte("{broken"), 0600); err != nil {
		t.Fatal(err)
	}
	_, err := NewStore(path).Snapshot(time.Now())
	if err == nil || !strings.Contains(err.Error(), "invalid JSON") {
		t.Fatalf("expected parsing error, got %v", err)
	}
	b, _ := os.ReadFile(path)
	if string(b) != "{broken" {
		t.Fatal("corrupted file unexpectedly overwritten")
	}
}
