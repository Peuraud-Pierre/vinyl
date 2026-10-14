package widgetupdate

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeEngine struct {
	observed Observed
	update   func(ctx context.Context) error
}

func (f *fakeEngine) Observe(context.Context) Observed { return f.observed }
func (f *fakeEngine) Update(ctx context.Context) error { return f.update(ctx) }

func collect() (func(Status), <-chan Status) {
	ch := make(chan Status, 16)
	return func(s Status) { ch <- s }, ch
}

func next(t *testing.T, ch <-chan Status) Status {
	t.Helper()
	select {
	case s := <-ch:
		return s
	case <-time.After(2 * time.Second):
		t.Fatal("aucun état émis")
		return Status{}
	}
}

func TestCheckEmitsOnChangeOnly(t *testing.T) {
	emit, ch := collect()
	e := &fakeEngine{observed: Observed{Installed: true, Expiry: testNow.Add(-time.Hour), Now: testNow}}
	m := NewManager(e, emit)
	m.Check(context.Background())
	if s := next(t, ch); s != (Status{State: StateAvailable, Reason: "expired", Message: "Signature du widget expirée"}) {
		t.Fatalf("état = %+v", s)
	}
	m.Check(context.Background())
	if len(ch) != 0 {
		t.Fatal("pas de nouvel événement si rien ne change")
	}
	e.observed = Observed{Installed: true, Now: testNow}
	m.Check(context.Background())
	if s := next(t, ch); s.State != StateNone {
		t.Fatalf("état = %+v", s)
	}
}

func TestStartSuccess(t *testing.T) {
	emit, ch := collect()
	e := &fakeEngine{observed: Observed{Installed: true, Now: testNow}, update: func(context.Context) error { return nil }}
	m := NewManager(e, emit)
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	if s := next(t, ch); s.State != StateRunning {
		t.Fatalf("état = %+v", s)
	}
	if s := next(t, ch); s.State != StateNone {
		t.Fatalf("état = %+v", s)
	}
}

func TestStartBusy(t *testing.T) {
	emit, ch := collect()
	release := make(chan struct{})
	e := &fakeEngine{update: func(context.Context) error { <-release; return nil }}
	m := NewManager(e, emit)
	m.Start()
	next(t, ch)
	if err := m.Start(); !errors.Is(err, ErrBusy) {
		t.Fatalf("err = %v, want ErrBusy", err)
	}
	m.Check(context.Background())
	if m.Status().State != StateRunning {
		t.Fatal("Check ne doit pas écraser une mise à jour en cours")
	}
	close(release)
	next(t, ch)
}

func TestErrorPersistsWhileReasonsRemain(t *testing.T) {
	emit, ch := collect()
	e := &fakeEngine{
		observed: Observed{Installed: true, Expiry: testNow.Add(-time.Hour), Now: testNow},
		update:   func(context.Context) error { return &UserError{Msg: "Installe XcodeGen : brew install xcodegen"} },
	}
	m := NewManager(e, emit)
	m.Start()
	next(t, ch)
	if s := next(t, ch); s != (Status{State: StateError, Message: "Installe XcodeGen : brew install xcodegen"}) {
		t.Fatalf("état = %+v", s)
	}
	m.Check(context.Background())
	if m.Status().State != StateError {
		t.Fatal("l'erreur doit rester affichée tant que la mise à jour est nécessaire")
	}
}
