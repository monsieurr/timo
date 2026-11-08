package timo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Store serializes read/modify/write transactions across Go processes and goroutines.
// The lock file persists; the OS releases the lock if a process crashes.
type Store struct {
	Path string
	mu   sync.Mutex
}

func DefaultPath() (string, error) {
	if p := os.Getenv("TIMO_DATA_FILE"); p != "" {
		return filepath.Abs(p)
	}
	if p := os.Getenv("POMO_DATA_FILE"); p != "" {
		return filepath.Abs(p)
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	// Prefer a timo file when present. Otherwise, continue using the legacy
	// pomo path so existing history is preserved without copying while an old
	// process may still have the JSON file open.
	current := filepath.Join(dir, "timo", "data.json")
	if _, err := os.Stat(current); err == nil || !errors.Is(err, os.ErrNotExist) {
		return current, nil
	}
	legacy := filepath.Join(dir, "pomo", "data.json")
	if _, err := os.Stat(legacy); err == nil {
		return legacy, nil
	}
	return current, nil
}
func NewStore(path string) *Store { return &Store{Path: path} }

func (s *Store) read() (State, error) {
	b, err := os.ReadFile(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return Defaults(), nil
	}
	if err != nil {
		return State{}, fmt.Errorf("read data: %w", err)
	}
	var v State
	if err := json.Unmarshal(b, &v); err != nil {
		return State{}, fmt.Errorf("invalid JSON in %s: %w (file left untouched)", s.Path, err)
	}
	if err := Validate(v); err != nil {
		return State{}, fmt.Errorf("invalid data in %s: %w (file left untouched)", s.Path, err)
	}
	if v.History == nil {
		v.History = []Session{}
	}
	return v, nil
}

func (s *Store) write(v State) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	dir := filepath.Dir(s.Path)
	f, err := os.CreateTemp(dir, ".timo-*.tmp")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err != nil {
		f.Close()
		return err
	}
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(name, s.Path); err != nil {
		return fmt.Errorf("replace JSON: %w", err)
	}
	return nil
}

// Update provides exactly one atomic read/settle/mutate/write transaction.
// Settling an expired timer is committed even if the requested action is invalid.
func (s *Store) Update(now time.Time, fn func(*State) error) (State, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(s.Path), 0700); err != nil {
		return State{}, err
	}
	lock, err := os.OpenFile(s.Path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return State{}, err
	}
	defer lock.Close()
	if err := lockFile(lock); err != nil {
		return State{}, fmt.Errorf("acquire file lock: %w", err)
	}
	defer unlockFile(lock)
	v, err := s.read()
	if err != nil {
		return State{}, err
	}
	settled := v.Settle(now)
	if fn != nil {
		if err := fn(&v); err != nil {
			if settled {
				if saveErr := s.write(v); saveErr != nil {
					return State{}, saveErr
				}
			}
			return v, err
		}
	}
	if fn != nil || settled {
		if err := s.write(v); err != nil {
			return State{}, err
		}
	} else if _, err := os.Stat(s.Path); errors.Is(err, os.ErrNotExist) {
		if err := s.write(v); err != nil {
			return State{}, err
		}
	}
	return v, nil
}
func (s *Store) Snapshot(now time.Time) (State, error) { return s.Update(now, nil) }
