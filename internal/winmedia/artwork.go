package winmedia

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"vinyl/internal/player"
)

// Pochette absente : nouvel essai après artworkRetry, au plus artworkTries fois (Spotify publie
// souvent le titre avant la pochette) ; jamais une lecture à chaque sondage.
const (
	artworkTries = 3
	artworkRetry = 2 * time.Second
)

type artworkState struct {
	url   string
	tries int
	next  time.Time // pas de nouvel essai avant
}

// artwork : pochettes écrites une fois par morceau dans le dossier servi sous /artwork/.
type artwork struct {
	dir  string
	mu   sync.Mutex
	done map[string]*artworkState // trackID → état
}

func (a *artwork) url(trackID string, s Session, now time.Time) string {
	if a.dir == "" {
		return ""
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	st := a.done[trackID]
	if st == nil {
		st = &artworkState{}
		a.done[trackID] = st
	}
	if st.url != "" || st.tries >= artworkTries || now.Before(st.next) {
		return st.url
	}
	st.tries++
	st.next = now.Add(artworkRetry)
	if b, err := s.Thumbnail(); err == nil && len(b) > 0 {
		file := player.ArtworkFileName(trackID)
		if os.MkdirAll(a.dir, 0o755) == nil && os.WriteFile(filepath.Join(a.dir, file), b, 0o644) == nil {
			st.url = "/artwork/" + file
		}
	}
	return st.url
}
