package player

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const (
	resyncEvery = 2 * time.Second
	scriptLimit = 2 * time.Second
)

type Snapshot struct {
	NowPlaying NowPlaying `json:"nowPlaying"`
	Problem    string     `json:"problem"`
	ProblemApp string     `json:"problemApp"`
}

type Watcher struct {
	players []Player
	mode    func() string
	emit    func(Snapshot)
	now     func() time.Time
	wake    chan struct{}
	force   atomic.Bool // posé par Wake : le prochain Tick émet même sans changement (ex. après un Seek)

	// Utilisés uniquement depuis Tick (une seule goroutine).
	prevState    map[string]State
	playingSince map[string]time.Time
	onPlaying    func(NowPlaying)

	mu          sync.Mutex
	active      string
	last        Snapshot
	lastEmit    time.Time
	emittedOnce bool
}

func NewWatcher(players []Player, mode func() string, emit func(Snapshot), now func() time.Time) *Watcher {
	return &Watcher{
		players: players, mode: mode, emit: emit, now: now,
		wake:         make(chan struct{}, 1),
		prevState:    map[string]State{},
		playingSince: map[string]time.Time{},
	}
}

func (w *Watcher) Player(name string) Player {
	for _, p := range w.players {
		if p.Name() == name {
			return p
		}
	}
	return nil
}

func (w *Watcher) Active() Player {
	w.mu.Lock()
	name := w.active
	w.mu.Unlock()
	if name == "" {
		return nil
	}
	return w.Player(name)
}

func (w *Watcher) Current() Snapshot {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.last
}

func (w *Watcher) Wake() {
	w.force.Store(true)
	select {
	case w.wake <- struct{}{}:
	default:
	}
}

func (w *Watcher) Run(ctx context.Context, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		w.Tick(ctx)
		select {
		case <-ctx.Done():
			return
		case <-t.C:
		case <-w.wake:
		}
	}
}

func (w *Watcher) Tick(ctx context.Context) {
	candidates := w.players
	if mode := w.mode(); mode != "auto" {
		if p := w.Player(mode); p != nil {
			candidates = []Player{p}
		}
	}

	now := w.now()
	results := map[string]NowPlaying{}
	permissionApp := ""

	w.mu.Lock()
	active, last := w.active, w.last
	w.mu.Unlock()

	for _, p := range candidates {
		cctx, cancel := context.WithTimeout(ctx, scriptLimit)
		np, err := p.NowPlaying(cctx)
		cancel()
		name := p.Name()
		switch {
		case errors.Is(err, ErrNotRunning):
			delete(w.prevState, name)
			continue
		case errors.Is(err, ErrNotAuthorized):
			if permissionApp == "" {
				permissionApp = name
			}
			continue
		case err != nil:
			// Erreur passagère (timeout…) : on garde le dernier état connu de la source active.
			if name == active && last.Problem == "" {
				results[name] = last.NowPlaying
			}
			continue
		}
		results[name] = np
		if np.State == StatePlaying && w.prevState[name] != StatePlaying {
			w.playingSince[name] = now
		}
		w.prevState[name] = np.State
	}

	chosen := ""
	for _, p := range candidates {
		np, ok := results[p.Name()]
		if ok && np.State == StatePlaying && (chosen == "" || w.playingSince[p.Name()].After(w.playingSince[chosen])) {
			chosen = p.Name()
		}
	}
	if chosen == "" {
		if _, ok := results[active]; ok {
			chosen = active
		}
	}
	if chosen == "" {
		for _, p := range candidates {
			if _, ok := results[p.Name()]; ok {
				chosen = p.Name()
				break
			}
		}
	}

	var snap Snapshot
	switch {
	case chosen != "":
		snap.NowPlaying = results[chosen]
	case permissionApp != "":
		snap.Problem, snap.ProblemApp = "permission", permissionApp
	default:
		snap.Problem = "no-player"
	}

	w.mu.Lock()
	w.active = chosen
	forced := w.force.Swap(false)
	send := forced || !w.emittedOnce || differs(w.last, snap) || now.Sub(w.lastEmit) >= resyncEvery
	w.last = snap
	if send {
		w.lastEmit = now
		w.emittedOnce = true
	}
	w.mu.Unlock()

	if w.onPlaying != nil && snap.Problem == "" && snap.NowPlaying.State == StatePlaying {
		w.onPlaying(snap.NowPlaying)
	}
	if send {
		w.emit(snap)
	}
}

// SetOnPlaying enregistre un rappel appelé à chaque sondage où la source active joue (à brancher avant Run).
func (w *Watcher) SetOnPlaying(f func(NowPlaying)) { w.onPlaying = f }

func differs(a, b Snapshot) bool {
	return a.Problem != b.Problem || a.ProblemApp != b.ProblemApp ||
		a.NowPlaying.Source != b.NowPlaying.Source || a.NowPlaying.TrackID != b.NowPlaying.TrackID ||
		a.NowPlaying.State != b.NowPlaying.State || a.NowPlaying.ArtworkURL != b.NowPlaying.ArtworkURL
}
