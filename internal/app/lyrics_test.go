package app

import (
	"context"
	"errors"
	"testing"
	"time"

	"vinyl/internal/lyrics"
	"vinyl/internal/player"
)

func TestLyricsCachedPerTrackButNotErrors(t *testing.T) {
	s := New(nil, player.NewWatcher(nil, func() string { return "auto" }, func(player.Snapshot) {}, time.Now), nil)
	calls := 0
	fail := true
	s.lyricsFind = func(context.Context, string, string, string, int64) (lyrics.Lyrics, error) {
		calls++
		if fail {
			return lyrics.Lyrics{}, errors.New("hors ligne")
		}
		return lyrics.Lyrics{Synced: []lyrics.Line{{TimeMs: 1000, Text: "ligne inventée"}}}, nil
	}
	if _, err := s.Lyrics("spotify", "spotify:track:1", "Titre", "Artiste", "Album", 180000); err == nil {
		t.Fatal("erreur attendue")
	}
	fail = false
	for range 2 {
		l, err := s.Lyrics("spotify", "spotify:track:1", "Titre", "Artiste", "Album", 180000)
		if err != nil || len(l.Synced) != 1 {
			t.Fatalf("got %+v, %v", l, err)
		}
	}
	if calls != 2 {
		t.Errorf("%d recherches, 2 attendues (l'erreur n'est pas gardée, le résultat oui)", calls)
	}
}
