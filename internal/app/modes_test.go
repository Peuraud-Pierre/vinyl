package app

import (
	"context"
	"testing"
	"time"

	"vinyl/internal/history"
	"vinyl/internal/player"
)

// fakeModesPlayer : lecteur actif qui connaît l'aléatoire et la répétition.
type fakeModesPlayer struct {
	shuffle    bool
	repeat     player.RepeatMode
	modesCalls int
	played     []string
	playErr    error
	nexts      int
}

func (f *fakeModesPlayer) Name() string { return "applemusic" }
func (f *fakeModesPlayer) NowPlaying(context.Context) (player.NowPlaying, error) {
	return player.NowPlaying{Source: "applemusic", State: player.StatePlaying, TrackID: "t"}, nil
}
func (f *fakeModesPlayer) PlayPause(context.Context) error           { return nil }
func (f *fakeModesPlayer) Next(context.Context) error                { f.nexts++; return nil }
func (f *fakeModesPlayer) Previous(context.Context) error            { return nil }
func (f *fakeModesPlayer) Seek(context.Context, time.Duration) error { return nil }
func (f *fakeModesPlayer) PlayItem(_ context.Context, id string) error {
	f.played = append(f.played, id)
	return f.playErr
}
func (f *fakeModesPlayer) SetShuffle(_ context.Context, on bool) error { f.shuffle = on; return nil }
func (f *fakeModesPlayer) SetRepeat(_ context.Context, m player.RepeatMode) error {
	f.repeat = m
	return nil
}
func (f *fakeModesPlayer) PlaybackModes(context.Context) (bool, player.RepeatMode, error) {
	f.modesCalls++
	return f.shuffle, f.repeat, nil
}

func serviceWith(p player.Player) *VinylService {
	w := player.NewWatcher([]player.Player{p}, func() string { return "auto" }, func(player.Snapshot) {}, time.Now)
	w.Tick(context.Background())
	return New(nil, w, nil)
}

// Boutons de la platine : l'état de l'aléatoire se lit sans charger la file d'attente.
func TestModesReportsShuffleAndRepeat(t *testing.T) {
	p := &fakeModesPlayer{shuffle: true, repeat: player.RepeatAll}
	m, err := serviceWith(p).Modes()
	if err != nil {
		t.Fatal(err)
	}
	if !m.Shuffle || m.Repeat != "all" {
		t.Errorf("modes = %+v", m)
	}
}

func TestModesWithoutPlayer(t *testing.T) {
	w := player.NewWatcher(nil, func() string { return "auto" }, func(player.Snapshot) {}, time.Now)
	if _, err := New(nil, w, nil).Modes(); err == nil {
		t.Error("sans lecteur actif, une erreur est attendue")
	}
}

func TestHistoryListsThePile(t *testing.T) {
	s := New(nil, player.NewWatcher(nil, func() string { return "auto" }, func(player.Snapshot) {}, time.Now), nil)
	if l := s.History(); l == nil || len(l) != 0 {
		t.Errorf("sans historique : liste vide attendue, obtenu %v", l)
	}
	h := history.NewStore(t.TempDir() + "/h.json")
	h.Playing(history.Entry{Source: "applemusic", TrackID: "1"})
	h.Playing(history.Entry{Source: "applemusic", TrackID: "2"})
	s.SetHistory(h, nil)
	if l := s.History(); len(l) != 1 || l[0].TrackID != "1" {
		t.Errorf("pile = %v", l)
	}
}

// Lecteur qui sait retenir sa playlist, jouer un morceau seul puis reprendre.
type fakeResumer struct {
	fakeModesPlayer
	saved   player.PlaybackContext
	alone   []string
	resumed []player.PlaybackContext
}

func (f *fakeResumer) SaveContext(context.Context) (player.PlaybackContext, error) {
	return f.saved, nil
}
func (f *fakeResumer) PlayAlone(_ context.Context, id string) error {
	f.alone = append(f.alone, id)
	return f.playErr
}
func (f *fakeResumer) UpcomingFrom(_ context.Context, playlistID, trackID string, limit int, wrap bool) ([]player.Track, error) {
	return []player.Track{{ID: trackID, Title: "repris"}, {ID: "x", Title: "suivant"}}, nil
}
func (f *fakeResumer) Resume(_ context.Context, c player.PlaybackContext) error {
	f.resumed = append(f.resumed, c)
	return nil
}

func pileOf(t *testing.T, ids ...string) *history.Store {
	h := history.NewStore(t.TempDir() + "/h.json")
	for _, id := range ids {
		h.Playing(history.Entry{Source: "applemusic", TrackID: id})
	}
	return h
}

func playing(id string, pos, dur int64) player.Snapshot {
	return player.Snapshot{NowPlaying: player.NowPlaying{Source: "applemusic", TrackID: id, State: player.StatePlaying, PositionMs: pos, DurationMs: dur}}
}

// Playlist en cours, au format des éléments de la bibliothèque.
func TestCurrentPlaylistAppleMusic(t *testing.T) {
	p := &fakeResumer{saved: player.PlaybackContext{PlaylistID: "PL", TrackID: "4"}}
	s := serviceWith(p)
	if got, err := s.CurrentPlaylist("applemusic"); err != nil || got != "playlist:PL" {
		t.Errorf("%q, %v", got, err)
	}
	p.saved.PlaylistID = ""
	if got, _ := s.CurrentPlaylist("applemusic"); got != "" {
		t.Errorf("sans playlist : %q", got)
	}
	if got, _ := s.CurrentPlaylist("spotify"); got != "" { // pas de catalogue Spotify branché
		t.Errorf("spotify : %q", got)
	}
}
