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
}

func (f *fakeModesPlayer) Name() string { return "applemusic" }
func (f *fakeModesPlayer) NowPlaying(context.Context) (player.NowPlaying, error) {
	return player.NowPlaying{Source: "applemusic", State: player.StatePlaying, TrackID: "t"}, nil
}
func (f *fakeModesPlayer) PlayPause(context.Context) error             { return nil }
func (f *fakeModesPlayer) Next(context.Context) error                  { return nil }
func (f *fakeModesPlayer) Previous(context.Context) error              { return nil }
func (f *fakeModesPlayer) Seek(context.Context, time.Duration) error   { return nil }
func (f *fakeModesPlayer) PlayItem(context.Context, string) error      { return nil }
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
	s.SetHistory(h)
	if l := s.History(); len(l) != 1 || l[0].TrackID != "1" {
		t.Errorf("pile = %v", l)
	}
}
