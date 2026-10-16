package app

import (
	"context"
	"errors"
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
}

func (f *fakeModesPlayer) Name() string { return "applemusic" }
func (f *fakeModesPlayer) NowPlaying(context.Context) (player.NowPlaying, error) {
	return player.NowPlaying{Source: "applemusic", State: player.StatePlaying, TrackID: "t"}, nil
}
func (f *fakeModesPlayer) PlayPause(context.Context) error           { return nil }
func (f *fakeModesPlayer) Next(context.Context) error                { return nil }
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

// Comme Apple Music : le morceau repris de la pile est joué seul, la pile n'est pas vidée, et à la fin
// du morceau la playlist reprend là où elle en était.
func TestReplayThenResumeWhereThePlaylistWas(t *testing.T) {
	p := &fakeResumer{saved: player.PlaybackContext{PlaylistID: "PL", TrackID: "4", PositionSec: 61}}
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil) // pile 321, en cours 4
	if err := s.PlayFromHistory("applemusic", "2"); err != nil {
		t.Fatal(err)
	}
	if len(p.alone) != 1 || p.alone[0] != "2" {
		t.Fatalf("joué seul : %v", p.alone)
	}
	if l := s.History(); len(l) != 3 {
		t.Errorf("la pile n'est pas vidée : %v", l)
	}
	s.ObservePlayback(playing("2", 1000, 200000))
	s.ObservePlayback(playing("2", 198500, 200000)) // presque fini
	if len(p.resumed) != 0 {
		t.Fatal("pas de reprise avant la fin")
	}
	s.ObservePlayback(player.Snapshot{NowPlaying: player.NowPlaying{Source: "applemusic", State: player.StateStopped}})
	// Comme un clic dans l'Historique de Musique : la playlist reprend au morceau qui SUIVAIT l'interrompu.
	want := p.saved
	want.After = true
	if len(p.resumed) != 1 || p.resumed[0] != want {
		t.Fatalf("reprise : %v", p.resumed)
	}
	s.ObservePlayback(playing("4", 61000, 300000))
	if len(p.resumed) != 1 {
		t.Error("une seule reprise")
	}
}

func TestResumeWhenTheReplayedSongLoopsOrMusicMovesOn(t *testing.T) {
	for name, after := range map[string]player.Snapshot{
		"répétition":      playing("2", 400, 200000),
		"morceau suivant": playing("99", 300, 180000),
	} {
		p := &fakeResumer{saved: player.PlaybackContext{PlaylistID: "PL", TrackID: "4"}}
		s := serviceWith(p)
		s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil)
		s.PlayFromHistory("applemusic", "2")
		s.ObservePlayback(playing("2", 199000, 200000))
		s.ObservePlayback(after)
		if len(p.resumed) != 1 {
			t.Errorf("%s : reprise attendue", name)
		}
	}
}

// Si l'utilisateur change lui-même de morceau en cours de route, on ne lui impose pas la reprise.
func TestNoResumeWhenTheUserPicksSomethingElse(t *testing.T) {
	p := &fakeResumer{saved: player.PlaybackContext{PlaylistID: "PL", TrackID: "4"}}
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil)
	s.PlayFromHistory("applemusic", "2")
	s.ObservePlayback(playing("2", 30000, 200000))
	s.ObservePlayback(playing("77", 500, 200000))
	s.ObservePlayback(player.Snapshot{NowPlaying: player.NowPlaying{Source: "applemusic", State: player.StateStopped}})
	if len(p.resumed) != 0 {
		t.Errorf("aucune reprise attendue : %v", p.resumed)
	}
}

func TestReplayFailureReported(t *testing.T) {
	p := &fakeResumer{}
	p.playErr = errors.New("Musique ne répond pas")
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2"), nil)
	if err := s.PlayFromHistory("applemusic", "1"); err == nil {
		t.Error("erreur attendue")
	}
	s.ObservePlayback(player.Snapshot{NowPlaying: player.NowPlaying{Source: "applemusic", State: player.StateStopped}})
	if len(p.resumed) != 0 {
		t.Error("pas de reprise après un échec")
	}
}

// Lecteur sans reprise (Spotify) : le morceau est lancé directement.
func TestReplayWithoutResumeSupport(t *testing.T) {
	p := &fakeModesPlayer{}
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2"), nil)
	if err := s.PlayFromHistory("applemusic", "1"); err != nil || len(p.played) != 1 || p.played[0] != "track:1" {
		t.Errorf("lancé : %v (%v)", p.played, err)
	}
}

func TestQueueDuringReplayShowsThePlaylistToResume(t *testing.T) {
	p := &fakeResumer{saved: player.PlaybackContext{PlaylistID: "PL", TrackID: "4"}}
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil)
	if err := s.PlayFromHistory("applemusic", "2"); err != nil {
		t.Fatal(err)
	}
	q, err := s.Queue()
	if err != nil {
		t.Fatal(err)
	}
	// La playlist reprendra au morceau qui suivait l'interrompu (« 4 » n'est pas repris).
	if len(q.Upcoming) != 1 || q.Upcoming[0].ID != "x" {
		t.Errorf("file = %+v", q.Upcoming)
	}
}
