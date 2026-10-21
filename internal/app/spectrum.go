package app

import (
	"context"
	"log"
	"sync"
	"time"

	"vinyl/internal/audiotap"
)

// Tap : capture du son de sortie (audiotap), remplacée dans les tests.
type Tap interface {
	Start() error
	Stop()
	Read() ([]float32, uint64)
}

type systemTap struct{}

func (systemTap) Start() error              { return audiotap.Start() }
func (systemTap) Stop()                     { audiotap.Stop() }
func (systemTap) Read() ([]float32, uint64) { return audiotap.Read() }

// spectrum diffuse les bandes de fréquences du son capté (événement « spectrum »), tant que la platine
// en a besoin : rien n'est capté quand la musique est en pause ou la platine fermée.
type spectrum struct {
	mu       sync.Mutex
	tap      Tap
	emit     func([]float32)
	interval time.Duration
	cancel   context.CancelFunc
	done     chan struct{}
}

func newSpectrum(tap Tap, emit func([]float32)) *spectrum {
	return &spectrum{tap: tap, emit: emit, interval: 33 * time.Millisecond}
}

func (sp *spectrum) start() error {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.cancel != nil {
		return nil
	}
	if err := sp.tap.Start(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	sp.cancel, sp.done = cancel, make(chan struct{})
	go sp.run(ctx, sp.done)
	return nil
}

// run envoie chaque nouvelle analyse, au plus une par intervalle.
func (sp *spectrum) run(ctx context.Context, done chan struct{}) {
	defer close(done)
	t := time.NewTicker(sp.interval)
	defer t.Stop()
	var last uint64
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			if b, seq := sp.tap.Read(); seq != last {
				last = seq
				sp.emit(b)
			}
		}
	}
}

func (sp *spectrum) stop() {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	if sp.cancel == nil {
		return
	}
	sp.cancel()
	<-sp.done
	sp.cancel, sp.done = nil, nil
	sp.tap.Stop()
}

// SpectrumAvailable : l'égaliseur peut suivre le vrai son (Mac, macOS 14.2 et plus).
func (s *VinylService) SpectrumAvailable() bool { return audiotap.Available() }

// StartSpectrum lance la capture du son pour l'égaliseur (la première fois, macOS demande l'autorisation).
func (s *VinylService) StartSpectrum() error {
	if !s.store.Get().EqAudio {
		return nil
	}
	err := s.spec.start()
	if err != nil {
		log.Printf("égaliseur : %v", err)
	}
	return err
}

// StopSpectrum arrête la capture du son.
func (s *VinylService) StopSpectrum() { s.spec.stop() }
