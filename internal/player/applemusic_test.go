package player

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type fakeRunner struct {
	outputs []string // une sortie par appel, dans l'ordre
	errs    []error
	calls   []fakeCall
}

type fakeCall struct {
	script string
	args   []string
}

func (f *fakeRunner) Run(_ context.Context, script string, args ...string) (string, error) {
	i := len(f.calls)
	f.calls = append(f.calls, fakeCall{script, args})
	var out string
	var err error
	if i < len(f.outputs) {
		out = f.outputs[i]
	}
	if i < len(f.errs) {
		err = f.errs[i]
	}
	return out, err
}

func join(fields ...string) string { return strings.Join(fields, Sep) }

func TestAppleMusicNowPlayingParsesFrenchLocale(t *testing.T) {
	r := &exportRunner{
		nowPlaying: join("playing", "ABC123", `Don't Stop "Live"`, "Beyoncé", "Album", "213,5", "12,25", "1"),
		exportOut:  "OK", content: []byte("\x89PNG"),
	}
	m := NewAppleMusic(r, t.TempDir())
	np, err := m.NowPlaying(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := NowPlaying{Source: "applemusic", TrackID: "ABC123", Title: `Don't Stop "Live"`, Artist: "Beyoncé",
		Album: "Album", DurationMs: 213500, PositionMs: 12250, State: StatePlaying,
		ArtworkURL: "/artwork/" + ArtworkFileName("ABC123")}
	if np != want {
		t.Errorf("got %+v\nwant %+v", np, want)
	}
	if r.exports != 1 {
		t.Errorf("un export de pochette attendu, obtenu %d", r.exports)
	}
}

func TestAppleMusicArtworkExportedOncePerTrack(t *testing.T) {
	dir := t.TempDir()
	r := &exportRunner{nowPlaying: join("playing", "ABC123", "T", "A", "B", "200", "1", "1"), exportOut: "OK", content: []byte("x")}
	m := NewAppleMusic(r, dir)
	m.NowPlaying(context.Background())
	m.NowPlaying(context.Background())
	if r.exports != 1 {
		t.Errorf("la pochette ne doit être exportée qu'une fois, obtenu %d", r.exports)
	}
}

func TestAppleMusicNoArtwork(t *testing.T) {
	r := &fakeRunner{outputs: []string{join("paused", "X", "T", "A", "B", "missing value", "0", "0")}}
	np, err := NewAppleMusic(r, t.TempDir()).NowPlaying(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if np.ArtworkURL != "" || np.DurationMs != 0 || np.State != StatePaused {
		t.Errorf("got %+v", np)
	}
}

func TestAppleMusicNotRunningAndStopped(t *testing.T) {
	m := NewAppleMusic(&fakeRunner{outputs: []string{"NOTRUNNING"}}, t.TempDir())
	if _, err := m.NowPlaying(context.Background()); err != ErrNotRunning {
		t.Errorf("attendu ErrNotRunning, obtenu %v", err)
	}
	m = NewAppleMusic(&fakeRunner{outputs: []string{"stopped"}}, t.TempDir())
	np, err := m.NowPlaying(context.Background())
	if err != nil || np.State != StateStopped || np.Source != "applemusic" {
		t.Errorf("got %+v, %v", np, err)
	}
}

func TestAppleMusicMalformedOutput(t *testing.T) {
	m := NewAppleMusic(&fakeRunner{outputs: []string{"playing" + Sep + "x"}}, t.TempDir())
	if _, err := m.NowPlaying(context.Background()); err == nil {
		t.Error("une sortie tronquée doit renvoyer une erreur")
	}
}

func TestAppleMusicSeekUsesIntegerSecondsInArgv(t *testing.T) {
	r := &fakeRunner{}
	NewAppleMusic(r, t.TempDir()).Seek(context.Background(), 93700*time.Millisecond)
	if len(r.calls) != 1 || r.calls[0].args[0] != "93" {
		t.Errorf("got %+v", r.calls)
	}
}

func TestAppleMusicPlayItemRouting(t *testing.T) {
	cases := []struct {
		id       string
		wantArgs []string
		wantIn   string
	}{
		{"playlist:P1", []string{"P1"}, "first playlist whose persistent ID"},
		{"track:T1", []string{"T1"}, "persistent ID"},
		{"album:Mon \"Album\"" + Sep + "Artiste", []string{`Mon "Album"`, "Artiste"}, "Vinyl — Album"},
	}
	for _, c := range cases {
		r := &fakeRunner{}
		if err := NewAppleMusic(r, t.TempDir()).PlayItem(context.Background(), c.id); err != nil {
			t.Fatalf("%s: %v", c.id, err)
		}
		got := r.calls[0]
		if strings.Join(got.args, "|") != strings.Join(c.wantArgs, "|") || !strings.Contains(got.script, c.wantIn) {
			t.Errorf("%s: args=%q script contient %q ? %v", c.id, got.args, c.wantIn, strings.Contains(got.script, c.wantIn))
		}
	}
	if err := NewAppleMusic(&fakeRunner{}, t.TempDir()).PlayItem(context.Background(), "inconnu:1"); err == nil {
		t.Error("un ID inconnu doit renvoyer une erreur")
	}
}

// exportRunner simule Musique : lecture du morceau puis export de la pochette dans le fichier argv[0].
type exportRunner struct {
	nowPlaying string
	exportOut  string // "OK" ou "CHANGED"
	content    []byte
	exports    int
}

func (r *exportRunner) Run(_ context.Context, script string, args ...string) (string, error) {
	if script == musicExportArtworkScript {
		r.exports++
		if r.exportOut == "OK" {
			os.WriteFile(args[0], r.content, 0o644)
		}
		return r.exportOut, nil
	}
	return r.nowPlaying, nil
}

func TestArtworkNotSavedWhenTrackChangedDuringExport(t *testing.T) {
	dir := t.TempDir()
	r := &exportRunner{nowPlaying: join("playing", "OLD", "T", "A", "B", "200", "1", "1"), exportOut: "CHANGED"}
	np, err := NewAppleMusic(r, dir).NowPlaying(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if np.ArtworkURL != "" {
		t.Errorf("pochette d'un autre morceau : ne doit pas être utilisée, got %q", np.ArtworkURL)
	}
	if _, err := os.Stat(filepath.Join(dir, ArtworkFileName("OLD"))); err == nil {
		t.Error("aucun fichier ne doit être enregistré sous l'ID de l'ancien morceau")
	}
}

func TestEmptyArtworkFileIsReexported(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ArtworkFileName("X")), nil, 0o644) // export précédent raté
	r := &exportRunner{nowPlaying: join("playing", "X", "T", "A", "B", "200", "1", "1"), exportOut: "OK", content: []byte("\x89PNG")}
	np, _ := NewAppleMusic(r, dir).NowPlaying(context.Background())
	if r.exports != 1 || np.ArtworkURL == "" {
		t.Errorf("un fichier vide doit être réexporté : exports=%d url=%q", r.exports, np.ArtworkURL)
	}
	if data, _ := os.ReadFile(filepath.Join(dir, ArtworkFileName("X"))); string(data) != "\x89PNG" {
		t.Errorf("contenu final = %q", data)
	}
}

func TestPlayListPassesIDsInOrderThroughArgv(t *testing.T) {
	r := &fakeRunner{}
	err := NewAppleMusic(r, t.TempDir()).PlayList(context.Background(), []string{"-e", "B", "C"}, "B", true)
	if err != nil {
		t.Fatal(err)
	}
	got := r.calls[0]
	if strings.Join(got.args, "|") != "B|1|-e|B|C" || !strings.Contains(got.script, "Vinyl — Album") {
		t.Errorf("args = %q", got.args)
	}
}

func TestPlayInPlaylistArgs(t *testing.T) {
	r := &fakeRunner{}
	NewAppleMusic(r, t.TempDir()).PlayInPlaylist(context.Background(), "P1", "", false)
	if strings.Join(r.calls[0].args, "|") != "P1||0" {
		t.Errorf("args = %q", r.calls[0].args)
	}
}

func TestMusicPlaybackModes(t *testing.T) {
	r := &fakeRunner{outputs: []string{join("true", "all")}}
	sh, rep, err := NewAppleMusic(r, t.TempDir()).PlaybackModes(context.Background())
	if err != nil || !sh || rep != RepeatAll {
		t.Errorf("got %v %v %v", sh, rep, err)
	}
	r = &fakeRunner{outputs: []string{join("false", "bizarre")}}
	_, rep, _ = NewAppleMusic(r, t.TempDir()).PlaybackModes(context.Background())
	if rep != RepeatOff {
		t.Errorf("valeur inconnue → off, got %v", rep)
	}
}

func TestMusicSetRepeatAndShuffleArgs(t *testing.T) {
	r := &fakeRunner{}
	m := NewAppleMusic(r, t.TempDir())
	m.SetRepeat(context.Background(), RepeatOne)
	m.SetShuffle(context.Background(), true)
	if r.calls[0].args[0] != "one" || r.calls[1].args[0] != "1" {
		t.Errorf("args = %+v", r.calls)
	}
	if err := m.SetRepeat(context.Background(), "jamais"); err == nil {
		t.Error("mode inconnu → erreur")
	}
}

func TestMusicUpcomingAtEndOfPlaylist(t *testing.T) {
	r := &fakeRunner{outputs: []string{"5" + Sep + "5" + "\x1e"}}
	ts, err := NewAppleMusic(r, t.TempDir()).Upcoming(context.Background(), 30, false)
	if err != nil || len(ts) != 0 {
		t.Errorf("got %v %v", ts, err)
	}
	if strings.Join(r.calls[0].args, "|") != "30|0" {
		t.Errorf("args = %q", r.calls[0].args)
	}
}

func TestMusicUpcomingParsesTracks(t *testing.T) {
	out := "2" + Sep + "4" + "\x1e" + join("X", "Trois", "A", "200", "Colors - Single") + "\x1e" + join("Y", "Quatre", "B", "180,5", "Discovery") + "\x1e"
	ts, _ := NewAppleMusic(&fakeRunner{outputs: []string{out}}, t.TempDir()).Upcoming(context.Background(), 30, true)
	if len(ts) != 2 || ts[1].Title != "Quatre" || ts[1].DurationMs != 180500 || ts[1].Album != "Discovery" {
		t.Errorf("got %+v", ts)
	}
}

func TestMusicUpcomingNoCurrentPlaylist(t *testing.T) {
	ts, err := NewAppleMusic(&fakeRunner{outputs: []string{"NONE"}}, t.TempDir()).Upcoming(context.Background(), 30, false)
	if err != nil || ts != nil {
		t.Errorf("radio/flux : liste vide sans erreur, got %v %v", ts, err)
	}
}

func TestPlaylistScriptsFindSubscriptionPlaylists(t *testing.T) {
	for name, script := range map[string]string{"PlayItem": musicPlayPlaylistScript, "PlayInPlaylist": musicPlayInPlaylistScript} {
		if strings.Contains(script, "user playlist whose persistent ID") {
			t.Errorf("%s : doit chercher parmi toutes les playlists, pas seulement les « user playlist »", name)
		}
	}
}

// Pile des derniers écoutés : on revient en arrière dans la playlist en cours (la suite continue
// après), et on ne lance le titre seul depuis la bibliothèque que s'il n'y est pas.
func TestReplayInCurrentPlaylist(t *testing.T) {
	r := &fakeRunner{outputs: []string{"PLAYLIST"}}
	if err := NewAppleMusic(r, t.TempDir()).ReplayInContext(context.Background(), "ABC123"); err != nil {
		t.Fatal(err)
	}
	c := r.calls[0]
	if len(c.args) != 1 || c.args[0] != "ABC123" {
		t.Errorf("identifiant transmis par argv : %v", c.args)
	}
	for _, want := range []string{"current playlist", "library playlist 1", `is not running`} {
		if !strings.Contains(c.script, want) {
			t.Errorf("script sans %q", want)
		}
	}
	if strings.Index(c.script, "current playlist") > strings.Index(c.script, "library playlist 1") {
		t.Error("la playlist en cours doit être essayée avant la bibliothèque")
	}
}

func TestReplayInContextNotRunning(t *testing.T) {
	r := &fakeRunner{outputs: []string{"NOTRUNNING"}}
	if err := NewAppleMusic(r, t.TempDir()).ReplayInContext(context.Background(), "X"); err != ErrNotRunning {
		t.Errorf("err = %v", err)
	}
}
