package winmedia

import (
	"context"
	"errors"
	"testing"
	"time"

	"vinyl/internal/player"
)

type fakeSession struct {
	id        string
	info      Info
	calls     []string
	seeks     []time.Duration
	thumb     []byte
	thumbErr  error
	thumbRead int
}

func (f *fakeSession) AppID() string       { return f.id }
func (f *fakeSession) Info() (Info, error) { return f.info, nil }
func (f *fakeSession) Thumbnail() ([]byte, error) {
	f.thumbRead++
	return f.thumb, f.thumbErr
}
func (f *fakeSession) TogglePlayPause() error { f.calls = append(f.calls, "playpause"); return nil }
func (f *fakeSession) Next() error            { f.calls = append(f.calls, "next"); return nil }
func (f *fakeSession) Previous() error        { f.calls = append(f.calls, "previous"); return nil }
func (f *fakeSession) Seek(d time.Duration) error {
	f.seeks = append(f.seeks, d)
	return nil
}
func (f *fakeSession) SetShuffle(on bool) error {
	f.info.Shuffle = on
	return nil
}
func (f *fakeSession) SetRepeat(m player.RepeatMode) error {
	f.info.Repeat = m
	return nil
}

type fakeSessions []Session

func (f fakeSessions) List() ([]Session, error) { return f, nil }

var t0 = time.Unix(1_000_000, 0)

func song(title string) Info {
	return Info{Title: title, Artist: "A", Album: "B", Position: 10 * time.Second, Duration: 200 * time.Second,
		Status: player.StatePlaying, UpdatedAt: t0}
}

func TestNotRunningWithoutSession(t *testing.T) {
	p := NewPlayer(SpotifySource, fakeSessions{&fakeSession{id: "Chrome", info: song("x")}}, func() time.Time { return t0 })
	if _, err := p.NowPlaying(context.Background()); !errors.Is(err, player.ErrNotRunning) {
		t.Errorf("err = %v", err)
	}
}

// Spotify classique et Spotify du Microsoft Store ; Apple Music pour Windows.
func TestSourcesAreRecognised(t *testing.T) {
	for id, src := range map[string]Source{
		"Spotify.exe": SpotifySource,
		"SpotifyAB.SpotifyMusic_zpdnekdrzrea0!Spotify": SpotifySource,
		"AppleInc.AppleMusicWin_nzyj5cx40ttqa!App":     AppleMusicSource,
	} {
		if !src.Match(id) {
			t.Errorf("%s non reconnu comme %s", id, src.Name)
		}
	}
	if SpotifySource.Match("AppleInc.AppleMusicWin_x!App") || AppleMusicSource.Match("Spotify.exe") {
		t.Error("confusion entre les sources")
	}
}

func TestNowPlaying(t *testing.T) {
	s := &fakeSession{id: "Spotify.exe", info: song("Creep")}
	p := NewPlayer(SpotifySource, fakeSessions{s}, func() time.Time { return t0.Add(3 * time.Second) })
	np, err := p.NowPlaying(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if np.Source != "spotify" || np.Title != "Creep" || np.State != player.StatePlaying || np.DurationMs != 200000 {
		t.Errorf("np = %+v", np)
	}
	if np.PositionMs != 13000 { // relevé à 10 s, 3 s plus tôt
		t.Errorf("position = %d", np.PositionMs)
	}
	if np.TrackID == "" || np.TrackID != TrackID("Creep", "A", "B") || TrackID("Creep", "A", "B") == TrackID("Creep", "A", "C") {
		t.Errorf("TrackID = %q", np.TrackID)
	}
}

func TestPositionNeverExceedsDuration(t *testing.T) {
	s := &fakeSession{id: "Spotify.exe", info: song("Creep")}
	p := NewPlayer(SpotifySource, fakeSessions{s}, func() time.Time { return t0.Add(time.Hour) })
	if np, _ := p.NowPlaying(context.Background()); np.PositionMs != 200000 {
		t.Errorf("position = %d", np.PositionMs)
	}
	s.info.Status = player.StatePaused // en pause : pas d'interpolation
	if np, _ := p.NowPlaying(context.Background()); np.PositionMs != 10000 {
		t.Errorf("position en pause = %d", np.PositionMs)
	}
}

// Publicité, changement de morceau : pas de titre → ni pile ni compteur.
func TestUntitledSessionIsStopped(t *testing.T) {
	s := &fakeSession{id: "Spotify.exe", info: Info{Status: player.StatePlaying}}
	np, err := NewPlayer(SpotifySource, fakeSessions{s}, time.Now).NowPlaying(context.Background())
	if err != nil || np.TrackID != "" || np.State != player.StateStopped {
		t.Errorf("np = %+v err = %v", np, err)
	}
}

func TestCommandsAndModes(t *testing.T) {
	s := &fakeSession{id: "AppleInc.AppleMusicWin_x!App", info: song("x")}
	p := NewPlayer(AppleMusicSource, fakeSessions{s}, time.Now)
	ctx := context.Background()
	p.PlayPause(ctx)
	p.Next(ctx)
	p.Previous(ctx)
	p.Seek(ctx, 42*time.Second)
	if len(s.calls) != 3 || s.calls[2] != "previous" || len(s.seeks) != 1 || s.seeks[0] != 42*time.Second {
		t.Errorf("calls %v seeks %v", s.calls, s.seeks)
	}
	p.SetShuffle(ctx, true)
	p.SetRepeat(ctx, player.RepeatAll)
	if sh, rep, err := p.PlaybackModes(ctx); !sh || rep != player.RepeatAll || err != nil {
		t.Errorf("modes %v %v %v", sh, rep, err)
	}
	if err := p.PlayItem(ctx, "album:x"); err == nil {
		t.Error("PlayItem : non disponible sous Windows")
	}
}
