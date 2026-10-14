package settings

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestLoadMissingFileGivesDefaults(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "settings.json"))
	if err := s.Load(); err != nil {
		t.Fatal(err)
	}
	if s.Get() != Defaults() {
		t.Errorf("got %+v", s.Get())
	}
}

func TestSaveThenLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "settings.json")
	s := NewStore(path)
	v := Defaults()
	v.Texture, v.Accent, v.Source = "yolk", "#ff6b6b", "spotify"
	if err := s.Save(v); err != nil {
		t.Fatal(err)
	}
	s2 := NewStore(path)
	if err := s2.Load(); err != nil {
		t.Fatal(err)
	}
	if s2.Get() != v {
		t.Errorf("got %+v want %+v", s2.Get(), v)
	}
}

func TestCorruptFileFallsBackToDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte("{pas du json"), 0o644)
	s := NewStore(path)
	if err := s.Load(); err != nil {
		t.Fatalf("un fichier corrompu ne doit pas bloquer le démarrage : %v", err)
	}
	if s.Get() != Defaults() {
		t.Errorf("got %+v", s.Get())
	}
}

func TestPartialOrInvalidValuesAreNormalized(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	os.WriteFile(path, []byte(`{"texture":"paillettes","accent":"rouge","source":"deezer","crackle":false}`), 0o644)
	s := NewStore(path)
	s.Load()
	got, d := s.Get(), Defaults()
	if got.Texture != d.Texture || got.Accent != d.Accent || got.Source != d.Source || got.Crackle != false {
		t.Errorf("got %+v", got)
	}
	if got.DiscColors != d.DiscColors {
		t.Errorf("couleurs manquantes → défaut, got %v", got.DiscColors)
	}
}

func TestUpdateAppliesOnlyGivenFields(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "settings.json"))
	src := "spotify"
	if _, err := s.Update(Patch{Source: &src}); err != nil {
		t.Fatal(err)
	}
	off := false
	got, err := s.Update(Patch{Crackle: &off})
	if err != nil {
		t.Fatal(err)
	}
	if got.Source != "spotify" || got.Crackle {
		t.Errorf("une mise à jour partielle ne doit pas écraser les autres champs : %+v", got)
	}
}

func TestConcurrentUpdatesLoseNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	s := NewStore(path)
	var wg sync.WaitGroup
	src, tex, acc := "spotify", "yolk", "#ff6b6b"
	for _, p := range []Patch{{Source: &src}, {Texture: &tex}, {Accent: &acc}} {
		wg.Add(1)
		go func(p Patch) {
			defer wg.Done()
			if _, err := s.Update(p); err != nil {
				t.Error(err)
			}
		}(p)
	}
	wg.Wait()
	s2 := NewStore(path)
	s2.Load()
	if g := s2.Get(); g.Source != src || g.Texture != tex || g.Accent != acc {
		t.Errorf("mises à jour simultanées perdues : %+v", g)
	}
}
