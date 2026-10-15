package catalog

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

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

// Images des playlists (mosaïque comme dans Musique) : VINYL_LIVE=1 VINYL_ART_DIR=… pour les garder.
func TestLivePlaylistArtwork(t *testing.T) {
	if os.Getenv("VINYL_LIVE") == "" {
		t.Skip("définir VINYL_LIVE=1")
	}
	dir := os.Getenv("VINYL_ART_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	m := NewMusicLibrary(player.OSAScript{}, NewITunes(), dir)
	p, err := m.Library(context.Background(), "playlists", 0)
	if err != nil {
		t.Skip(err)
	}
	for _, it := range p.Items {
		start := time.Now()
		u, err := m.Artwork(context.Background(), it.ID)
		if err != nil {
			t.Errorf("%s : %v", it.Title, err)
		}
		t.Logf("%-22s %-60s %v", it.Title, u, time.Since(start).Round(time.Millisecond))
	}
}
