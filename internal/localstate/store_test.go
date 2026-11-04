package localstate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestHistoryPersistsDeletesAndRespectsPreference(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "state")
	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if err = s.Record("https://example.org/a", "Title", now); err != nil {
		t.Fatal(err)
	}
	if err = s.Record("https://example.org/a", "New title", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if len(s.Snapshot().Entries) != 1 {
		t.Fatal("history must deduplicate URLs")
	}
	second, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if second.Snapshot().Entries[0].Title != "New title" {
		t.Fatal("history did not persist")
	}
	if err = s.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if err = s.Record("https://example.org/b", "Ignored", now); err != nil {
		t.Fatal(err)
	}
	if len(s.Snapshot().Entries) != 1 {
		t.Fatal("disabled history saved a visit")
	}
	if err = s.Delete(s.Snapshot().Entries[0].ID); err != nil {
		t.Fatal(err)
	}
	if len(s.Snapshot().Entries) != 0 {
		t.Fatal("delete retained entry")
	}
	if err = s.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	_ = s.Record("https://example.org/c", "Third", now)
	if err = s.Clear(); err != nil {
		t.Fatal(err)
	}
	if len(s.Snapshot().Entries) != 0 {
		t.Fatal("clear retained entries")
	}
	for _, file := range []string{dir, filepath.Join(dir, "history.json")} {
		st, e := os.Stat(file)
		if e != nil {
			t.Fatal(e)
		}
		want := os.FileMode(0600)
		if file == dir {
			want = 0700
		}
		if st.Mode().Perm() != want {
			t.Errorf("permissions %o want %o", st.Mode().Perm(), want)
		}
	}
	b, _ := os.ReadFile(filepath.Join(dir, "history.json"))
	if strings.Contains(string(b), "example.org") {
		t.Fatal("clear did not persist")
	}
}
func TestSnapshotCannotMutateStore(t *testing.T) {
	s, _ := Open(t.TempDir())
	_ = s.Record("https://example.org", "Title", time.Now())
	copy := s.Snapshot()
	copy.Entries[0].Title = "Corrupted"
	if s.Snapshot().Entries[0].Title != "Title" {
		t.Fatal("snapshot aliases internal state")
	}
}

func TestIndependentStoresDoNotLoseEachOthersWrites(t *testing.T) {
	dir := t.TempDir()
	a, _ := Open(dir)
	b, _ := Open(dir)
	if err := a.Record("https://example.org/a", "A", time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := b.Record("https://example.org/b", "B", time.Now()); err != nil {
		t.Fatal(err)
	}
	if got := len(a.Snapshot().Entries); got != 2 {
		t.Fatalf("independent process lost history: %d entries", got)
	}
	if err := b.Clear(); err != nil {
		t.Fatal(err)
	}
	if len(a.Snapshot().Entries) != 0 {
		t.Fatal("cleared history still visible to another process")
	}
}
