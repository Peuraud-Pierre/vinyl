package player

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestSpotifyNowPlaying(t *testing.T) {
	r := &fakeRunner{outputs: []string{
		join("playing", "spotify:track:42", "Titre", "Artiste", "Album", "185000", "12,5", "https://i.scdn.co/image/abc"),
	}}
	np, err := NewSpotify(r).NowPlaying(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := NowPlaying{Source: "spotify", TrackID: "spotify:track:42", Title: "Titre", Artist: "Artiste", Album: "Album",
		DurationMs: 185000, PositionMs: 12500, State: StatePlaying, ArtworkURL: "https://i.scdn.co/image/abc"}
	if np != want {
		t.Errorf("got %+v\nwant %+v", np, want)
	}
}

func TestSpotifyAdHasNoDurationOrArtwork(t *testing.T) {
	r := &fakeRunner{outputs: []string{join("playing", "spotify:ad:1", "Pub", "", "", "0", "3", "missing value")}}
	np, _ := NewSpotify(r).NowPlaying(context.Background())
	if np.DurationMs != 0 || np.ArtworkURL != "" {
		t.Errorf("got %+v", np)
	}
}

func TestSpotifyNotRunning(t *testing.T) {
	if _, err := NewSpotify(&fakeRunner{outputs: []string{"NOTRUNNING"}}).NowPlaying(context.Background()); err != ErrNotRunning {
		t.Errorf("got %v", err)
	}
}

func TestSpotifySeekAndPlayItem(t *testing.T) {
	r := &fakeRunner{}
	s := NewSpotify(r)
	s.Seek(context.Background(), 61*time.Second)
	s.PlayItem(context.Background(), "spotify:album:9")
	if r.calls[0].args[0] != "61" || r.calls[1].args[0] != "spotify:album:9" {
		t.Errorf("got %+v", r.calls)
	}
	if err := s.PlayItem(context.Background(), "playlist:1"); err == nil {
		t.Error("une URI non Spotify doit être refusée")
	}
}

func TestSpotifyPlayInContext(t *testing.T) {
	r := &fakeRunner{}
	s := NewSpotify(r)
	if err := s.PlayInContext(context.Background(), "spotify:album:9", "spotify:track:3", true); err != nil {
		t.Fatal(err)
	}
	if strings.Join(r.calls[0].args, "|") != "spotify:album:9|spotify:track:3|1" {
		t.Errorf("args = %q", r.calls[0].args)
	}
	if err := s.PlayInContext(context.Background(), "playlist:1", "", false); err == nil {
		t.Error("contexte non Spotify → erreur")
	}
}

func TestSpotifyModes(t *testing.T) {
	r := &fakeRunner{outputs: []string{join("false", "true")}}
	s := NewSpotify(r)
	sh, rep, _ := s.PlaybackModes(context.Background())
	if sh || rep != RepeatAll {
		t.Errorf("got %v %v", sh, rep)
	}
	s.SetRepeat(context.Background(), RepeatOne)
	if r.calls[1].args[0] != "1" {
		t.Errorf("« un seul » n'existe pas sur Spotify : traité comme « tout », args = %q", r.calls[1].args)
	}
}
