package stats

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var day = func() time.Time { return time.Date(2026, 10, 14, 15, 0, 0, 0, time.Local) }

func TestFirstLaunchStartsAtZeroToday(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "stats.json"), day)
	s.Load()
	if g := s.Get(); g.Count != 0 || g.Since != "2026.10.14" {
		t.Errorf("got %+v", g)
	}
}

func TestPlayedCountsNewTracksOnlyAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	s := NewStore(path, day)
	s.Load()
	if _, changed := s.Played("A"); !changed {
		t.Fatal("premier morceau : doit compter")
	}
	if _, changed := s.Played("A"); changed {
		t.Error("même morceau (pause puis reprise) : ne doit pas recompter")
	}
	if _, changed := s.Played(""); changed {
		t.Error("ID vide : ne compte pas")
	}
	got, _ := s.Played("B")
	if got.Count != 2 {
		t.Errorf("count = %d", got.Count)
	}
	s2 := NewStore(path, func() time.Time { return day().AddDate(0, 0, 3) })
	s2.Load()
	if g := s2.Get(); g.Count != 2 || g.Since != "2026.10.14" {
		t.Errorf("relu depuis le disque : %+v", g)
	}
}

func TestCorruptFileRestartsAtZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	os.WriteFile(path, []byte("{oups"), 0o644)
	s := NewStore(path, day)
	s.Load()
	if g := s.Get(); g.Count != 0 || g.Since != "2026.10.14" {
		t.Errorf("got %+v", g)
	}
}

func TestConcurrentPlaysAreAllCounted(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "stats.json"), day)
	s.Load()
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() { defer wg.Done(); s.Played(string(rune('a' + i))) }()
	}
	wg.Wait()
	if g := s.Get(); g.Count != 20 {
		t.Errorf("20 morceaux tous différents : count = %d", g.Count)
	}
}
