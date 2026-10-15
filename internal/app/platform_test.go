package app

import (
	"errors"
	"runtime"
	"testing"
	"time"

	"vinyl/internal/catalog"
	"vinyl/internal/player"
)

// Sous Windows, pas de bibliothèque Apple Music : le service doit le dire, pas planter.
func TestServiceWithoutAppleMusicLibrary(t *testing.T) {
	w := player.NewWatcher(nil, func() string { return "auto" }, func(player.Snapshot) {}, time.Now)
	s := New(nil, w, nil)
	if _, err := s.Library("applemusic", "albums", 0); !errors.Is(err, catalog.ErrUnsupported) {
		t.Errorf("Library : %v", err)
	}
	if u, err := s.ItemArtwork("applemusic", "album:X"); u != "" || err != nil {
		t.Errorf("ItemArtwork : %q %v", u, err)
	}
}

func TestPlatform(t *testing.T) {
	if p := New(nil, player.NewWatcher(nil, func() string { return "auto" }, func(player.Snapshot) {}, time.Now), nil).Platform(); p != runtime.GOOS {
		t.Errorf("Platform = %q", p)
	}
}
