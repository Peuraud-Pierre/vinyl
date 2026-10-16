package player

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

type stubPlayer struct {
	name string
	np   NowPlaying
	err  error
}

func (s *stubPlayer) Name() string { return s.name }
func (s *stubPlayer) NowPlaying(context.Context) (NowPlaying, error) {
	s.np.Source = s.name
	return s.np, s.err
}
func (s *stubPlayer) PlayPause(context.Context) error           { return nil }
func (s *stubPlayer) Next(context.Context) error                { return nil }
func (s *stubPlayer) Previous(context.Context) error            { return nil }
func (s *stubPlayer) Seek(context.Context, time.Duration) error { return nil }
func (s *stubPlayer) PlayItem(context.Context, string) error    { return nil }

type harness struct {
	am, sp  *stubPlayer
	mode    string
	now     time.Time
	emitted []Snapshot
	w       *Watcher
}

func newHarness() *harness {
	h := &harness{
		am:   &stubPlayer{name: "applemusic", err: ErrNotRunning},
		sp:   &stubPlayer{name: "spotify", err: ErrNotRunning},
		mode: "auto",
		now:  time.Unix(1000, 0),
	}
	h.w = NewWatcher([]Player{h.am, h.sp}, func() string { return h.mode },
		func(s Snapshot) { h.emitted = append(h.emitted, s) }, func() time.Time { return h.now })
	return h
}

func (h *harness) tick(advance time.Duration) {
	h.now = h.now.Add(advance)
	h.w.Tick(context.Background())
}

func playing(id string) NowPlaying {
	return NowPlaying{TrackID: id, State: StatePlaying, DurationMs: 1000}
}

func TestNoPlayerRunning(t *testing.T) {
	h := newHarness()
	h.tick(0)
	if len(h.emitted) != 1 || h.emitted[0].Problem != "no-player" || h.w.Active() != nil {
		t.Fatalf("got %+v", h.emitted)
	}
}

func TestPermissionProblem(t *testing.T) {
	h := newHarness()
	h.am.err = ErrNotAuthorized
	h.tick(0)
	if h.emitted[0].Problem != "permission" || h.emitted[0].ProblemApp != "applemusic" {
		t.Fatalf("got %+v", h.emitted[0])
	}
}

func TestAutoPicksPlayingSource(t *testing.T) {
	h := newHarness()
	h.am.err, h.am.np = nil, NowPlaying{TrackID: "a", State: StatePaused}
	h.sp.err, h.sp.np = nil, playing("s")
	h.tick(0)
	if got := h.emitted[len(h.emitted)-1].NowPlaying.Source; got != "spotify" {
		t.Fatalf("source = %s", got)
	}
}

func TestAutoBothPlayingLatestStartedWins(t *testing.T) {
	h := newHarness()
	h.sp.err, h.sp.np = nil, playing("s")
	h.tick(0)
	h.am.err, h.am.np = nil, playing("a")
	h.tick(500 * time.Millisecond)
	if h.w.Active().Name() != "applemusic" {
		t.Fatalf("active = %s", h.w.Active().Name())
	}
}

func TestPausedSourceStaysActive(t *testing.T) {
	h := newHarness()
	h.sp.err, h.sp.np = nil, playing("s")
	h.am.err, h.am.np = nil, NowPlaying{TrackID: "a", State: StatePaused}
	h.tick(0)
	h.sp.np.State = StatePaused
	h.tick(500 * time.Millisecond)
	if h.w.Active().Name() != "spotify" {
		t.Fatalf("la source en pause doit rester active, obtenu %s", h.w.Active().Name())
	}
}

func TestForcedModeIgnoresOtherPlayer(t *testing.T) {
	h := newHarness()
	h.mode = "applemusic"
	h.sp.err, h.sp.np = nil, playing("s")
	h.tick(0)
	if h.emitted[0].Problem != "no-player" {
		t.Fatalf("got %+v", h.emitted[0])
	}
}

func TestEmitsOnlyOnChangeOrEveryTwoSeconds(t *testing.T) {
	h := newHarness()
	h.am.err, h.am.np = nil, playing("a")
	h.tick(0)
	h.am.np.PositionMs = 500
	h.tick(500 * time.Millisecond)
	h.tick(500 * time.Millisecond)
	if len(h.emitted) != 1 {
		t.Fatalf("pas de changement : 1 émission attendue, obtenu %d", len(h.emitted))
	}
	h.tick(1000 * time.Millisecond) // 2 s depuis la dernière émission
	if len(h.emitted) != 2 {
		t.Fatalf("recalage à 2 s attendu, obtenu %d", len(h.emitted))
	}
	h.am.np.State = StatePaused
	h.tick(500 * time.Millisecond)
	if len(h.emitted) != 3 {
		t.Fatalf("changement d'état : émission attendue, obtenu %d", len(h.emitted))
	}
}

func TestTransientErrorKeepsLastSnapshot(t *testing.T) {
	h := newHarness()
	h.am.err, h.am.np = nil, playing("a")
	h.tick(0)
	h.am.err = errors.New("timeout")
	h.tick(500 * time.Millisecond)
	if h.w.Current().Problem != "" || h.w.Current().NowPlaying.TrackID != "a" {
		t.Fatalf("une erreur passagère ne doit pas effacer l'état : %+v", h.w.Current())
	}
}

func TestPlayerQuitWhilePlaying(t *testing.T) {
	h := newHarness()
	h.am.err, h.am.np = nil, playing("a")
	h.tick(0)
	h.am.err = ErrNotRunning
	h.tick(500 * time.Millisecond)
	if h.w.Current().Problem != "" {
		t.Fatalf("raté passager : %+v", h.w.Current()) // voir TestBriefLossKeepsLastState
	}
	h.tick(lostGrace)
	if h.w.Current().Problem != "no-player" {
		t.Fatalf("got %+v", h.w.Current())
	}
}

func TestWakeForcesEmissionAfterSeek(t *testing.T) {
	h := newHarness()
	h.am.err, h.am.np = nil, playing("a")
	h.tick(0)
	// L'utilisateur glisse le bras : même morceau, même état, nouvelle position.
	h.am.np.PositionMs = 600
	h.w.Wake()
	h.tick(300 * time.Millisecond)
	if len(h.emitted) != 2 || h.emitted[1].NowPlaying.PositionMs != 600 {
		t.Fatalf("après Wake, la nouvelle position doit être émise tout de suite : %+v", h.emitted)
	}
	h.tick(300 * time.Millisecond)
	if len(h.emitted) != 2 {
		t.Fatalf("le forçage ne vaut qu'une fois, obtenu %d émissions", len(h.emitted))
	}
}

func TestOnPlayingCalledOnlyWhilePlaying(t *testing.T) {
	h := newHarness()
	var seen []string
	h.w.SetOnPlaying(func(np NowPlaying) { seen = append(seen, np.TrackID) })
	h.am.err, h.am.np = nil, NowPlaying{TrackID: "a", State: StatePaused}
	h.tick(0)
	h.am.np.State = StatePlaying
	h.tick(500 * time.Millisecond)
	h.am.np.TrackID = "b"
	h.tick(500 * time.Millisecond)
	if len(seen) != 2 || seen[0] != "a" || seen[1] != "b" {
		t.Errorf("appels = %v", seen)
	}
}

// countingPlayer compte les sondages (accès concurrents : Run tourne dans sa goroutine).
type countingPlayer struct {
	stubPlayer
	mu sync.Mutex
	n  int
}

func (c *countingPlayer) NowPlaying(ctx context.Context) (NowPlaying, error) {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
	return c.stubPlayer.NowPlaying(ctx)
}
func (c *countingPlayer) count() int { c.mu.Lock(); defer c.mu.Unlock(); return c.n }

// Économie d'énergie : l'intervalle de sondage est redemandé à chaque tour (court près de la fin d'un
// morceau suivi, long à l'arrêt) ; un réveil (action de l'utilisateur) sonde aussitôt.
func TestRunAdaptiveInterval(t *testing.T) {
	p := &countingPlayer{stubPlayer: stubPlayer{name: "applemusic", err: ErrNotRunning}}
	w := NewWatcher([]Player{p}, func() string { return "auto" }, func(Snapshot) {}, time.Now)
	ctx, cancel := context.WithCancel(context.Background())
	var mu sync.Mutex
	interval := time.Hour
	go w.RunAdaptive(ctx, func() time.Duration { mu.Lock(); defer mu.Unlock(); return interval })
	time.Sleep(60 * time.Millisecond)
	if n := p.count(); n != 1 {
		t.Fatalf("intervalle long : un seul sondage attendu, %d", n)
	}
	w.Wake()
	time.Sleep(60 * time.Millisecond)
	if n := p.count(); n != 2 {
		t.Fatalf("réveil : sondage immédiat attendu, %d", n)
	}
	mu.Lock()
	interval = 10 * time.Millisecond
	mu.Unlock()
	w.Wake()
	time.Sleep(120 * time.Millisecond)
	cancel()
	if n := p.count(); n < 6 {
		t.Errorf("intervalle court : sondages rapprochés attendus, %d", n)
	}
}

// Musique ne répond pas une fois (pause, démarrage) : on garde le dernier état au lieu d'afficher
// « Lance Musique ou Spotify » ; seulement si ça dure, le problème s'affiche.
func TestBriefLossKeepsLastState(t *testing.T) {
	h := newHarness()
	h.am.err, h.am.np = nil, playing("A")
	h.tick(0)
	h.am.err = ErrNotRunning // raté passager
	h.tick(time.Second)
	if s := h.w.Current(); s.Problem != "" || s.NowPlaying.TrackID != "A" || h.w.Active() == nil {
		t.Fatalf("raté passager affiché : %+v", s)
	}
	h.am.err, h.am.np = nil, NowPlaying{TrackID: "A", State: StatePaused, DurationMs: 1000}
	h.tick(time.Second)
	if s := h.w.Current(); s.Problem != "" || s.NowPlaying.State != StatePaused {
		t.Fatalf("après reprise : %+v", s)
	}
	h.am.err = ErrNotRunning // Musique vraiment quitté
	h.tick(time.Second)
	h.tick(lostGrace)
	if s := h.w.Current(); s.Problem != "no-player" {
		t.Errorf("Musique quitté : %+v", s)
	}
}

// Pendant une réécoute par « précédent » (Vinyl fait défiler la file), les morceaux traversés ne sont ni
// lus ni affichés.
func TestHoldSkipsTicks(t *testing.T) {
	h := newHarness()
	h.am.err, h.am.np = nil, playing("A")
	h.tick(0)
	h.w.Hold(true)
	h.am.np = playing("traversé")
	h.tick(time.Second)
	if s := h.w.Current(); s.NowPlaying.TrackID != "A" || len(h.emitted) != 1 {
		t.Fatalf("morceau traversé vu : %+v", s)
	}
	h.w.Hold(false)
	h.am.np = playing("B")
	h.tick(time.Second)
	if h.w.Current().NowPlaying.TrackID != "B" {
		t.Error("le suivi doit reprendre")
	}
}
