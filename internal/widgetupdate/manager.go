package widgetupdate

import (
	"context"
	"errors"
	"sync"
	"time"
)

const (
	StateNone      = "none"
	StateAvailable = "available"
	StateRunning   = "running"
	StateError     = "error"
)

// Status est envoyé au frontend (événement « widgetupdate »).
type Status struct {
	State   string `json:"state"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

type Engine interface {
	Observe(ctx context.Context) Observed
	Update(ctx context.Context) error
}

var ErrBusy = errors.New("mise à jour du widget déjà en cours")

const updateTimeout = 10 * time.Minute

type Manager struct {
	engine Engine
	emit   func(Status)
	mu     sync.Mutex
	status Status
}

func NewManager(e Engine, emit func(Status)) *Manager {
	return &Manager{engine: e, emit: emit, status: Status{State: StateNone}}
}

func (m *Manager) Status() Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.status
}

// set change l'état et l'émet s'il a changé. keep permet de refuser le changement sous verrou.
func (m *Manager) set(s Status, keep func(cur Status) bool) {
	m.mu.Lock()
	if (keep != nil && keep(m.status)) || s == m.status {
		m.mu.Unlock()
		return
	}
	m.status = s
	m.mu.Unlock()
	m.emit(s)
}

// Check relève l'état du widget. Ne touche ni à une mise à jour en cours, ni à une erreur encore pertinente.
func (m *Manager) Check(ctx context.Context) {
	o := m.engine.Observe(ctx)
	reasons := Detect(o)
	next := Status{State: StateNone}
	if len(reasons) > 0 {
		next = Status{State: StateAvailable, Reason: string(reasons[0]), Message: Message(reasons[0], o.Expiry, o.Now)}
	}
	m.set(next, func(cur Status) bool {
		return cur.State == StateRunning || (cur.State == StateError && len(reasons) > 0)
	})
}

// Start lance la mise à jour en arrière-plan.
func (m *Manager) Start() error {
	m.mu.Lock()
	if m.status.State == StateRunning {
		m.mu.Unlock()
		return ErrBusy
	}
	s := Status{State: StateRunning, Message: "Mise à jour du widget…"}
	m.status = s
	m.mu.Unlock()
	m.emit(s)
	go m.run()
	return nil
}

func (m *Manager) run() {
	ctx, cancel := context.WithTimeout(context.Background(), updateTimeout)
	defer cancel()
	if err := m.engine.Update(ctx); err != nil {
		m.set(Status{State: StateError, Message: ErrorMessage(err)}, nil)
		return
	}
	m.set(Status{State: StateNone}, nil)
	m.Check(context.Background())
}

// Run vérifie tout de suite, puis à chaque intervalle, jusqu'à l'annulation de ctx.
func (m *Manager) Run(ctx context.Context, every time.Duration) {
	m.Check(ctx)
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			m.Check(ctx)
		}
	}
}
