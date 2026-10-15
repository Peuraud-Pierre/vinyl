package app

import (
	"testing"
	"time"

	"vinyl/internal/player"
)

func newBare() *VinylService {
	return New(nil, player.NewWatcher(nil, func() string { return "auto" }, func(player.Snapshot) {}, time.Now), nil)
}

func TestVersion(t *testing.T) {
	s := newBare()
	if v := s.Version(); v != "dev" {
		t.Errorf("par défaut : %q", v)
	}
	s.SetVersion("1.0.3")
	if v := s.Version(); v != "1.0.3" {
		t.Errorf("injectée : %q", v)
	}
}
