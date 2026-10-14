package player

import (
	"context"
	"os"
	"testing"
)

// Test d'intégration avec les vraies apps : VINYL_LIVE=1 go test ./internal/player/ -run Live -v
func TestLiveNowPlaying(t *testing.T) {
	if os.Getenv("VINYL_LIVE") == "" {
		t.Skip("définir VINYL_LIVE=1 pour interroger Musique et Spotify")
	}
	for _, p := range []Player{NewAppleMusic(OSAScript{}, t.TempDir()), NewSpotify(OSAScript{})} {
		np, err := p.NowPlaying(context.Background())
		if err != nil && err != ErrNotRunning && err != ErrNotAuthorized {
			t.Errorf("%s : %v", p.Name(), err)
		}
		t.Logf("%s : %+v (err=%v)", p.Name(), np, err)
		if m, ok := p.(Modes); ok {
			_, _, err := m.PlaybackModes(context.Background())
			if err != nil && err != ErrNotRunning && err != ErrNotAuthorized {
				t.Errorf("%s modes : %v", p.Name(), err)
			}
		}
	}
}
