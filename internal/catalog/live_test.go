package catalog

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"vinyl/internal/player"
)

// Tests d'intégration réels : VINYL_LIVE=1 go test ./internal/catalog/ -run Live -v
func TestLiveITunes(t *testing.T) {
	if os.Getenv("VINYL_LIVE") == "" {
		t.Skip("définir VINYL_LIVE=1")
	}
	p, err := NewITunes().Artist(context.Background(), "Daft Punk")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.TopTracks) == 0 || len(p.Albums) == 0 || p.ImageURL == "" {
		t.Errorf("page incomplète : %d titres, %d albums, image %q", len(p.TopTracks), len(p.Albums), p.ImageURL)
	}
	t.Logf("%s : %d titres, %d albums, ex. %q", p.Name, len(p.TopTracks), len(p.Albums), p.TopTracks[0].Title)
}

// Vérifie que les scripts compilent : seules les erreurs « non lancé » ou « non autorisé » sont acceptables.
func TestLiveMusicScriptsCompile(t *testing.T) {
	if os.Getenv("VINYL_LIVE") == "" {
		t.Skip("définir VINYL_LIVE=1")
	}
	m := NewMusicLibrary(player.OSAScript{}, NewITunes(), t.TempDir())
	for _, kind := range []string{"playlists", "albums", "artists"} {
		p, err := m.Library(context.Background(), kind, 0)
		if err != nil && !errors.Is(err, player.ErrNotAuthorized) && !errors.Is(err, player.ErrNotRunning) {
			t.Errorf("%s : %v", kind, err)
		}
		t.Logf("%s : %d éléments (err=%v)", kind, len(p.Items), err)
	}
}

// Les scripts de titres compilent ; une playlist introuvable (-1728/-1719) est acceptée.
func TestLiveMusicTracksScriptsCompile(t *testing.T) {
	if os.Getenv("VINYL_LIVE") == "" {
		t.Skip("définir VINYL_LIVE=1")
	}
	m := NewMusicLibrary(player.OSAScript{}, NewITunes(), t.TempDir())
	for _, id := range []string{"album:-e" + player.Sep + `"x"`, "playlist:0000000000000000"} {
		_, err := m.Tracks(context.Background(), id)
		if err != nil && !errors.Is(err, player.ErrNotAuthorized) && !errors.Is(err, player.ErrNotRunning) &&
			!strings.Contains(err.Error(), "(-1728)") && !strings.Contains(err.Error(), "(-1719)") {
			t.Errorf("%s : %v", id, err)
		}
	}
}
