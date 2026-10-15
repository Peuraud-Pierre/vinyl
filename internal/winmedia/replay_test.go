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
