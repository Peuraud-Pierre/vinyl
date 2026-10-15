package winmedia

import (
	"context"
	"testing"
	"time"

	"vinyl/internal/player"
)

// queueSession : une file de morceaux ; précédent / suivant déplacent la lecture.
type queueSession struct {
	fakeSession
	queue []string
	idx   int
}

func (q *queueSession) Info() (Info, error) { return song(q.queue[q.idx]), nil }
func (q *queueSession) Previous() error {
	if q.idx > 0 {
		q.idx--
	}
	return nil
}
func (q *queueSession) Next() error {
	if q.idx < len(q.queue)-1 {
		q.idx++
	}
	return nil
}

func replayPlayer(q *queueSession) *Player {
	p := NewPlayer(SpotifySource, fakeSessions{q}, time.Now, "")
	p.sleep = func(time.Duration) {}
	return p
}

func id(title string) string { return TrackID(title, "A", "B") }

func TestReplayBackFindsTheTrack(t *testing.T) {
	q := &queueSession{fakeSession: fakeSession{id: "Spotify.exe"}, queue: []string{"1", "2", "3", "4"}, idx: 3}
	taken, err := replayPlayer(q).ReplayBack(context.Background(), id("2"), 2, 61)
	if err != nil || taken != 2 || q.idx != 1 {
		t.Errorf("taken=%d idx=%d err=%v", taken, q.idx, err)
	}
}

// Pile décalée : le morceau est un cran plus loin.
func TestReplayBackOneStepFurther(t *testing.T) {
	q := &queueSession{fakeSession: fakeSession{id: "Spotify.exe"}, queue: []string{"1", "2", "3", "4"}, idx: 3}
	if taken, _ := replayPlayer(q).ReplayBack(context.Background(), id("1"), 2, 61); taken != 3 || q.idx != 0 {
		t.Errorf("taken=%d idx=%d", taken, q.idx)
	}
}

// Introuvable : retour au morceau de départ, à sa position.
func TestReplayBackNotFoundGoesBack(t *testing.T) {
	q := &queueSession{fakeSession: fakeSession{id: "Spotify.exe"}, queue: []string{"1", "2", "3", "4"}, idx: 3}
	taken, err := replayPlayer(q).ReplayBack(context.Background(), id("zz"), 1, 61)
	if err != nil || taken != 0 || q.idx != 3 {
		t.Fatalf("taken=%d idx=%d err=%v", taken, q.idx, err)
	}
	if len(q.seeks) != 1 || q.seeks[0] != 61*time.Second {
		t.Errorf("position non rétablie : %v", q.seeks)
	}
}

func TestJumpForward(t *testing.T) {
	q := &queueSession{fakeSession: fakeSession{id: "Spotify.exe"}, queue: []string{"1", "2", "3", "4"}, idx: 1}
	if err := replayPlayer(q).JumpForward(context.Background(), 2, id("4"), 30); err != nil {
		t.Fatal(err)
	}
	if q.idx != 3 || len(q.seeks) != 1 || q.seeks[0] != 30*time.Second {
		t.Errorf("idx=%d seeks=%v", q.idx, q.seeks)
	}
}

var _ player.BackReplayer = (*Player)(nil)

// Où en est la lecture, pour y revenir après une réécoute.
func TestSaveContext(t *testing.T) {
	q := &queueSession{fakeSession: fakeSession{id: "Spotify.exe"}, queue: []string{"1", "2"}, idx: 1}
	p := NewPlayer(SpotifySource, fakeSessions{q}, func() time.Time { return t0 }, "") // relevé à l'instant même
	c, err := p.SaveContext(context.Background())
	if err != nil || c.TrackID != id("2") || c.PositionSec != 10 {
		t.Errorf("contexte %+v err %v", c, err)
	}
}

// Comme Spotify / Apple Music sous Windows : « précédent » au-delà de 3 s relance d'abord le morceau.
type restartSession struct {
	queueSession
	pos time.Duration
}

func (r *restartSession) Info() (Info, error) {
	in := song(r.queue[r.idx])
	in.Position, in.Status = r.pos, player.StatePaused
	return in, nil
}
func (r *restartSession) Previous() error {
	if r.pos > 3*time.Second {
		r.pos = 0
		return nil
	}
	r.pos = 0
	return r.queueSession.Previous()
}
func (r *restartSession) Next() error { r.pos = 0; return r.queueSession.Next() }
func (r *restartSession) Seek(d time.Duration) error {
	r.pos = d
	return r.queueSession.Seek(d)
}

func restartPlayer(r *restartSession) *Player {
	p := NewPlayer(SpotifySource, fakeSessions{r}, func() time.Time { return t0 }, "")
	p.sleep = func(time.Duration) {}
	return p
}

// Réécoute puis retour : on revient exactement au morceau interrompu, même si le premier
// « précédent » n'a fait que le relancer.
func TestReplayAndReturnWithRestartOnPrevious(t *testing.T) {
	r := &restartSession{queueSession: queueSession{fakeSession: fakeSession{id: "Spotify.exe"}, queue: []string{"1", "2", "3", "4"}, idx: 3}, pos: 61 * time.Second}
	p := restartPlayer(r)
	taken, err := p.ReplayBack(context.Background(), id("3"), 1, 61)
	if err != nil || taken == 0 || r.idx != 2 {
		t.Fatalf("recul : taken=%d idx=%d err=%v", taken, r.idx, err)
	}
	if err := p.JumpForward(context.Background(), taken, id("4"), 61); err != nil {
		t.Fatal(err)
	}
	if r.idx != 3 || r.pos != 61*time.Second {
		t.Errorf("retour : idx=%d pos=%v (attendu le morceau 4 à 61 s)", r.idx, r.pos)
	}
}

// Introuvable : retour au morceau de départ, sans le dépasser.
func TestReplayNotFoundWithRestartOnPrevious(t *testing.T) {
	r := &restartSession{queueSession: queueSession{fakeSession: fakeSession{id: "Spotify.exe"}, queue: []string{"1", "2", "3", "4"}, idx: 3}, pos: 61 * time.Second}
	if taken, _ := restartPlayer(r).ReplayBack(context.Background(), id("zz"), 1, 61); taken != 0 || r.idx != 3 || r.pos != 61*time.Second {
		t.Errorf("taken=%d idx=%d pos=%v", taken, r.idx, r.pos)
	}
}
