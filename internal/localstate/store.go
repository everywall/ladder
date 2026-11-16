// Package localstate stores only reader history and preferences on this server.
package localstate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	ID       string    `json:"id"`
	URL      string    `json:"url"`
	Title    string    `json:"title"`
	ViewedAt time.Time `json:"viewedAt"`
}
type Snapshot struct {
	HistoryEnabled bool    `json:"historyEnabled"`
	Entries        []Entry `json:"entries"`
}
type Store struct {
	mu    sync.Mutex
	file  string
	state Snapshot
}

func DefaultDir() (string, error) {
	if s := os.Getenv("LADDER_STATE_DIR"); s != "" {
		return s, nil
	}
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "ladder"), nil
}
func Open(dir string) (*Store, error) {
	if dir == "" {
		return nil, fmt.Errorf("state directory is empty")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return nil, err
	}
	s := &Store{file: filepath.Join(dir, "history.json"), state: Snapshot{HistoryEnabled: true, Entries: []Entry{}}}
	b, err := os.ReadFile(s.file)
	if os.IsNotExist(err) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if len(b) > 2<<20 {
		return nil, fmt.Errorf("history file exceeds 2 MiB")
	}
	if err = json.Unmarshal(b, &s.state); err != nil {
		return nil, fmt.Errorf("read history: %w", err)
	}
	if len(s.state.Entries) > 100 {
		s.state.Entries = s.state.Entries[:100]
	}
	if s.state.Entries == nil {
		s.state.Entries = []Entry{}
	}
	if err = os.Chmod(s.file, 0600); err != nil {
		return nil, err
	}
	return s, nil
}
func (s *Store) reload() error {
	b, err := os.ReadFile(s.file)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if len(b) > 2<<20 {
		return fmt.Errorf("history file exceeds 2 MiB")
	}
	var next Snapshot
	if err = json.Unmarshal(b, &next); err != nil {
		return err
	}
	if len(next.Entries) > 100 {
		next.Entries = next.Entries[:100]
	}
	if next.Entries == nil {
		next.Entries = []Entry{}
	}
	s.state = next
	return nil
}
func (s *Store) SnapshotChecked() (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	lock, err := s.diskLock()
	if err != nil {
		return Snapshot{}, err
	}
	defer unlock(lock)
	if err = s.reload(); err != nil {
		return Snapshot{}, err
	}
	return Snapshot{HistoryEnabled: s.state.HistoryEnabled, Entries: append([]Entry{}, s.state.Entries...)}, nil
}
func (s *Store) Snapshot() Snapshot { snapshot, _ := s.SnapshotChecked(); return snapshot }
func (s *Store) mutate(fn func(*Snapshot)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	lock, err := s.diskLock()
	if err != nil {
		return err
	}
	defer unlock(lock)
	if err = s.reload(); err != nil {
		return err
	}
	next := Snapshot{HistoryEnabled: s.state.HistoryEnabled, Entries: append([]Entry{}, s.state.Entries...)}
	fn(&next)
	b, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(filepath.Dir(s.file), ".history-*")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if err = f.Chmod(0600); err == nil {
		_, err = f.Write(b)
	}
	if err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err = os.Rename(name, s.file); err != nil {
		return err
	}
	s.state = next
	return nil
}
func (s *Store) Record(rawURL, title string, now time.Time) error {
	return s.mutate(func(next *Snapshot) {
		if !next.HistoryEnabled {
			return
		}
		sum := sha256.Sum256([]byte(rawURL))
		id := hex.EncodeToString(sum[:12])
		entries := []Entry{{ID: id, URL: rawURL, Title: title, ViewedAt: now.UTC()}}
		for _, e := range next.Entries {
			if e.ID != id {
				entries = append(entries, e)
			}
		}
		if len(entries) > 100 {
			entries = entries[:100]
		}
		next.Entries = entries
	})
}
func (s *Store) Delete(id string) error {
	return s.mutate(func(next *Snapshot) {
		out := []Entry{}
		for _, e := range next.Entries {
			if e.ID != id {
				out = append(out, e)
			}
		}
		next.Entries = out
	})
}
func (s *Store) Clear() error { return s.mutate(func(next *Snapshot) { next.Entries = []Entry{} }) }
func (s *Store) SetEnabled(enabled bool) error {
	return s.mutate(func(next *Snapshot) { next.HistoryEnabled = enabled })
}
