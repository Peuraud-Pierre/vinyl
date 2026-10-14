package player

import (
	"context"
	"errors"
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
