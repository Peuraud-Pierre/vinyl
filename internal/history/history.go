// Package history garde les derniers morceaux écoutés, comme des disques posés en pile après les
// avoir lus : le plus récent au-dessus, sans doublon, au plus Max. Enregistré sur disque.
package history

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

// Max est le nombre de morceaux gardés sur la pile.
const Max = 7

type Entry struct {
	Source     string `json:"source"`
	TrackID    string `json:"trackId"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	ArtworkURL string `json:"artworkUrl"`
}

type file struct {
	Current Entry   `json:"current"`
	Recent  []Entry `json:"recent"`
}

type Store struct {
	path string
	mu   sync.Mutex
	f    file
}

func NewStore(path string) *Store { return &Store{path: path} }

// DefaultPath : ~/Library/Application Support/Vinyl/history.json.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Vinyl", "history.json"), nil
}

// Load lit la pile enregistrée ; absente ou illisible → pile vide.
func (s *Store) Load() {
	s.mu.Lock()
	defer s.mu.Unlock()
	var f file
	if data, err := os.ReadFile(s.path); err == nil && json.Unmarshal(data, &f) == nil {
		if len(f.Recent) > Max {
			f.Recent = f.Recent[:Max]
		}
		s.f = f
	}
}

// List renvoie la pile, le plus récent en premier.
func (s *Store) List() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Entry{}, s.f.Recent...)
}

func same(a, b Entry) bool { return a.Source == b.Source && a.TrackID == b.TrackID }

// Playing est appelé à chaque sondage où un morceau joue. Quand le morceau change, le précédent
// est posé sur la pile. Renvoie la pile et si elle a changé.
func (s *Store) Playing(e Entry) ([]Entry, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.TrackID == "" {
		return append([]Entry{}, s.f.Recent...), false
	}
	if same(e, s.f.Current) {
		if e != s.f.Current { // détails arrivés après coup (pochette)
			s.f.Current = e
			s.writeLocked()
		}
		return append([]Entry{}, s.f.Recent...), false
	}
	prev := s.f.Current
	s.f.Current = e
	recent := make([]Entry, 0, Max+1)
	if prev.TrackID != "" {
		recent = append(recent, prev)
	}
	for _, x := range s.f.Recent {
		// Sans doublon, et le morceau repris pour être écouté quitte la pile.
		if !same(x, prev) && !same(x, e) {
			recent = append(recent, x)
		}
	}
	if len(recent) > Max {
		recent = recent[:Max]
	}
	changed := !equal(recent, s.f.Recent)
	s.f.Recent = recent
	s.writeLocked()
	return append([]Entry{}, recent...), changed
}

func equal(a, b []Entry) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// writeLocked enregistre via un fichier temporaire. Appelant : verrou tenu.
func (s *Store) writeLocked() {
	data, _ := json.MarshalIndent(s.f, "", "  ")
	if os.MkdirAll(filepath.Dir(s.path), 0o755) != nil {
		return
	}
	tmp := s.path + ".tmp"
	if os.WriteFile(tmp, data, 0o644) == nil {
		os.Rename(tmp, s.path)
	}
}

// TakeFrom reprend un disque de la pile pour l'écouter : ceux posés au-dessus sont retirés et il
// devient le morceau en cours (le morceau qui jouait n'est pas reposé). Renvoie de quoi annuler, si
// la lecture échoue. Disque absent : rien ne change.
func (s *Store) TakeFrom(source, trackID string) (undo func()) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, x := range s.f.Recent {
		if x.Source == source && x.TrackID == trackID {
			saved := file{Current: s.f.Current, Recent: append([]Entry{}, s.f.Recent...)}
			s.f.Current = x
			s.f.Recent = append([]Entry{}, s.f.Recent[i+1:]...)
			s.writeLocked()
			return func() {
				s.mu.Lock()
				defer s.mu.Unlock()
				s.f = saved
				s.writeLocked()
			}
		}
	}
	return func() {}
}
