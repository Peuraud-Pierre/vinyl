package app

import (
	"errors"
	"sync"
	"testing"
	"time"
)

type fakeTap struct {
	mu      sync.Mutex
	running bool
	starts  int
	seq     uint64
	err     error
}

func (f *fakeTap) Start() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.running = true
	f.starts++
	return nil
}
func (f *fakeTap) Stop() { f.mu.Lock(); f.running = false; f.mu.Unlock() }
func (f *fakeTap) Read() ([]float32, uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return []float32{float32(f.seq)}, f.seq
}
func (f *fakeTap) analyse() { f.mu.Lock(); f.seq++; f.mu.Unlock() }

func TestSpectrumEmitsOnlyNewAnalyses(t *testing.T) {
	tap := &fakeTap{}
	var mu sync.Mutex
	var got []float32
	sp := newSpectrum(tap, func(b []float32) { mu.Lock(); got = append(got, b[0]); mu.Unlock() })
	sp.interval = time.Millisecond
	if err := sp.start(); err != nil {
		t.Fatal(err)
	}
	if err := sp.start(); err != nil || tap.starts != 1 {
		t.Fatalf("second start: err=%v starts=%d, want one capture", err, tap.starts)
	}
	time.Sleep(10 * time.Millisecond) // pas de nouvelle analyse : rien d'envoyé
	tap.analyse()
	time.Sleep(10 * time.Millisecond)
	tap.analyse()
	time.Sleep(10 * time.Millisecond)
	sp.stop()
	mu.Lock()
	defer mu.Unlock()
	if len(got) != 2 || got[0] != 1 || got[1] != 2 {
		t.Fatalf("emitted %v, want [1 2]", got)
	}
	if tap.running {
		t.Fatal("capture still running after stop")
	}
	sp.stop() // sans effet
}

func TestSpectrumStartError(t *testing.T) {
	tap := &fakeTap{err: errors.New("refusé")}
	sp := newSpectrum(tap, func([]float32) {})
	if err := sp.start(); err == nil {
		t.Fatal("want error")
	}
	sp.stop() // rien à arrêter
}
