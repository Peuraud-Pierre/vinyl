package winmedia

import (
	"os"
	"path/filepath"
	"sync"

	"vinyl/internal/player"
)

// artwork : pochettes écrites une fois par morceau dans le dossier servi sous /artwork/ ; un échec est
// retenu pour ne pas relire la miniature à chaque sondage.
type artwork struct {
	dir  string
	mu   sync.Mutex
	done map[string]string // trackID → URL ("" : pas de pochette)
}

func (a *artwork) url(trackID string, s Session) string {
	if a.dir == "" {
		return ""
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if u, ok := a.done[trackID]; ok {
		return u
	}
	u := ""
	if b, err := s.Thumbnail(); err == nil && len(b) > 0 {
		file := player.ArtworkFileName(trackID)
		if os.MkdirAll(a.dir, 0o755) == nil && os.WriteFile(filepath.Join(a.dir, file), b, 0o644) == nil {
			u = "/artwork/" + file
		}
	}
	a.done[trackID] = u
	return u
}
