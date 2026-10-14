// Package stats compte les morceaux lus depuis le premier lancement.
package stats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Snapshot struct {
	Count int64  `json:"count"`
	Since string `json:"since"` // AAAA.MM.JJ
}

type Store struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
	cur  Snapshot
	last string // dernier morceau compté
}

func NewStore(path string, now func() time.Time) *Store { return &Store{path: path, now: now} }

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir() // ~/Library/Application Support
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Vinyl", "stats.json"), nil
}

// Load lit le fichier ; absent ou illisible → 0 depuis aujourd'hui.
func (s *Store) Load() {
	s.mu.Lock()
	defer s.mu.Unlock()
	var v Snapshot
	if data, err := os.ReadFile(s.path); err == nil && json.Unmarshal(data, &v) == nil && v.Since != "" && v.Count >= 0 {
		s.cur = v
		return
	}
	s.cur = Snapshot{Since: s.now().Format("2006.01.02")}
	s.writeLocked()
}

func (s *Store) Get() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cur
}

// Played compte trackID s'il diffère du dernier morceau compté. Renvoie l'état et s'il a changé.
func (s *Store) Played(trackID string) (Snapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if trackID == "" || trackID == s.last {
		return s.cur, false
	}
	s.last = trackID
	s.cur.Count++
	s.writeLocked()
	return s.cur, true
}

// writeLocked enregistre l'état via un fichier temporaire. Appelant : verrou tenu.
func (s *Store) writeLocked() {
	data, _ := json.MarshalIndent(s.cur, "", "  ")
	if os.MkdirAll(filepath.Dir(s.path), 0o755) != nil {
		return
	}
	tmp := s.path + ".tmp"
	if os.WriteFile(tmp, data, 0o644) == nil {
		os.Rename(tmp, s.path)
	}
}
