package winmedia

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vinyl/internal/player"
)

func TestArtworkWrittenOnceAndServed(t *testing.T) {
	dir := t.TempDir()
	s := &fakeSession{id: "Spotify.exe", info: song("Creep"), thumb: []byte("jpeg")}
	p := NewPlayer(SpotifySource, fakeSessions{s}, time.Now, dir)
	for range 3 {
		np, _ := p.NowPlaying(context.Background())
		want := "/artwork/" + player.ArtworkFileName(np.TrackID)
		if np.ArtworkURL != want {
			t.Fatalf("ArtworkURL = %q", np.ArtworkURL)
		}
	}
	if s.thumbRead != 1 {
		t.Errorf("miniature lue %d fois (une seule attendue)", s.thumbRead)
	}
	b, err := os.ReadFile(filepath.Join(dir, player.ArtworkFileName(TrackID("Creep", "A", "B"))))
	if err != nil || string(b) != "jpeg" {
		t.Errorf("fichier : %q %v", b, err)
	}
}

// Miniature illisible : pas de pochette, pas de plantage, et pas de nouvel essai à chaque sondage.
func TestArtworkFailureIsRemembered(t *testing.T) {
	s := &fakeSession{id: "Spotify.exe", info: song("Creep"), thumbErr: errors.New("winrt")}
	p := NewPlayer(SpotifySource, fakeSessions{s}, time.Now, t.TempDir())
	for range 3 {
		if np, err := p.NowPlaying(context.Background()); err != nil || np.ArtworkURL != "" {
			t.Fatalf("np=%+v err=%v", np, err)
		}
	}
	if s.thumbRead != 1 {
		t.Errorf("miniature relue %d fois", s.thumbRead)
	}
}
