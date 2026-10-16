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
	for _, id := range []string{"inconnu:1", "album:X" + Sep + "Y"} { // album : joué titre par titre par Vinyl
		if err := NewAppleMusic(&fakeRunner{}, t.TempDir()).PlayItem(context.Background(), id); err == nil {
			t.Errorf("%s : erreur attendue", id)
		}
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

// Réécoute depuis la pile, comme dans Apple Music : on retient où en est la playlist, on joue le
// morceau seul (rien ne s'enchaîne derrière), puis on reprend la playlist là où elle en était.
func TestSaveContext(t *testing.T) {
	r := &fakeRunner{outputs: []string{join("PL1", "TR9", "83,6")}}
	c, err := NewAppleMusic(r, t.TempDir()).SaveContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if c != (PlaybackContext{PlaylistID: "PL1", TrackID: "TR9", PositionSec: 83}) {
		t.Errorf("contexte = %+v", c)
	}
	if !strings.Contains(r.calls[0].script, "current playlist") {
		t.Error("la playlist en cours doit être lue")
	}
}

func TestResumePlaysTheSavedTrackAtItsPosition(t *testing.T) {
	r := &fakeRunner{outputs: []string{"OK"}}
	err := NewAppleMusic(r, t.TempDir()).Resume(context.Background(), PlaybackContext{PlaylistID: "PL1", TrackID: "TR9", PositionSec: 83})
	if err != nil {
		t.Fatal(err)
	}
	c := r.calls[0]
	if strings.Join(c.args, "|") != "PL1|TR9|83" || !strings.Contains(c.script, "player position") {
		t.Errorf("argv %v", c.args)
	}
}

func TestPlayAloneNotRunning(t *testing.T) {
	r := &fakeRunner{outputs: []string{"NOTRUNNING"}}
	if err := NewAppleMusic(r, t.TempDir()).PlayAlone(context.Background(), "X"); err != ErrNotRunning {
		t.Errorf("err = %v", err)
	}
}

// Pendant une réécoute, la file affiche la suite de la playlist qui sera reprise, à partir du morceau
// interrompu (comme Apple Music), et non la playlist interne qui ne contient que le morceau réécouté.
func TestUpcomingFromSavedPlaylist(t *testing.T) {
	r := &fakeRunner{outputs: []string{"5" + Sep + "9\x1e" + join("T5", "Cinq", "A", "200", "Alb") + "\x1e" + join("T6", "Six", "A", "180", "Alb") + "\x1e"}}
	ts, err := NewAppleMusic(r, t.TempDir()).UpcomingFrom(context.Background(), "PL", "T5", 30, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(ts) != 2 || ts[0].ID != "T5" || ts[1].Title != "Six" {
		t.Errorf("file = %+v", ts)
	}
	if strings.Join(r.calls[0].args, "|") != "PL|T5|30|0" {
		t.Errorf("argv %v", r.calls[0].args)
	}
}

// Vinyl ne crée aucune playlist dans la bibliothèque de l'utilisateur.
func TestNoScriptCreatesAPlaylist(t *testing.T) {
	for _, f := range []string{"applemusic.go", "spotify.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(b), "make new") {
			t.Errorf("%s crée quelque chose dans la bibliothèque (« make new »)", f)
		}
	}
}

func TestPlayAloneUsesOnceWithoutPlaylist(t *testing.T) {
	r := &fakeRunner{outputs: []string{"OK"}}
	NewAppleMusic(r, t.TempDir()).PlayAlone(context.Background(), "ABC")
	if c := r.calls[0].script; !strings.Contains(c, "with once") || strings.Contains(c, "user playlist") {
		t.Error("le morceau doit être joué une fois, sans playlist")
	}
}

// Réécoute sans toucher à la file de Musique : « précédent » N fois, attente du changement de morceau
// (AutoMix, fondus), un ou deux morceaux de plus si la pile est décalée ; sinon retour où on était.
func TestReplayBackWalksThroughTheQueue(t *testing.T) {
	r := &fakeRunner{outputs: []string{"BACK" + Sep + "4"}}
	taken, err := NewAppleMusic(r, t.TempDir()).ReplayBack(context.Background(), "T2", 3, 61)
	if err != nil || taken != 4 {
		t.Fatalf("taken=%d err=%v", taken, err)
	}
	c := r.calls[0]
	if strings.Join(c.args, "|") != "T2|3|61" {
		t.Errorf("argv %v", c.args)
	}
	for _, want := range []string{"previous track", "next track", "player position", "delay"} {
		if !strings.Contains(c.script, want) {
			t.Errorf("script sans %q", want)
		}
	}
	if strings.Contains(c.script, "play (") || strings.Contains(c.script, "with once") {
		t.Error("jamais de nouveau « play » : la file de Musique ne doit pas être remplacée")
	}
	r = &fakeRunner{outputs: []string{"MISS"}}
	if taken, _ := NewAppleMusic(r, t.TempDir()).ReplayBack(context.Background(), "T2", 3, 61); taken != 0 {
		t.Error("morceau introuvable : 0 attendu")
	}
}

// Retour au morceau interrompu : « suivant » N fois puis position retrouvée.
func TestJumpForwardBackToTheInterruptedTrack(t *testing.T) {
	r := &fakeRunner{outputs: []string{"OK"}}
	if err := NewAppleMusic(r, t.TempDir()).JumpForward(context.Background(), 3, "T5", 61); err != nil {
		t.Fatal(err)
	}
	c := r.calls[0]
	if strings.Join(c.args, "|") != "3|T5|61" || !strings.Contains(c.script, "next track") || !strings.Contains(c.script, "player position") {
		t.Errorf("argv %v", c.args)
	}
}

// Économie d'énergie : une app qui n'est pas lancée n'est pas interrogée (aucun osascript lancé).
func TestNotRunningAppIsNotQueried(t *testing.T) {
	old := IsRunning
	defer func() { IsRunning = old }()
	IsRunning = func(string) bool { return false }
	r := &fakeRunner{}
	if _, err := NewAppleMusic(r, t.TempDir()).NowPlaying(context.Background()); err != ErrNotRunning || len(r.calls) != 0 {
		t.Errorf("Musique : err=%v, appels=%d", err, len(r.calls))
	}
	if _, err := NewSpotify(r).NowPlaying(context.Background()); err != ErrNotRunning || len(r.calls) != 0 {
		t.Errorf("Spotify : err=%v, appels=%d", err, len(r.calls))
	}
}

// Les sauts de morceaux se font son coupé, et le volume d'origine est toujours remis (même en cas d'erreur).
func TestAppleMusicSkipsAreMuted(t *testing.T) {
	r := &fakeRunner{outputs: []string{"MISS", "OK"}}
	m := NewAppleMusic(r, t.TempDir())
	m.ReplayBack(context.Background(), "T", 2, 0)
	m.JumpForward(context.Background(), 3, "", 0)
	for _, c := range r.calls {
		if !strings.Contains(c.script, "set sound volume to 0") || strings.Count(c.script, "set sound volume to vol") < 2 {
			t.Errorf("son non coupé ou non rétabli :\n%s", c.script)
		}
	}
}

// Reprise « au morceau suivant » : la playlist reprend au morceau qui suivait l'interrompu.
func TestAppleMusicResumeAfter(t *testing.T) {
	r := &fakeRunner{outputs: []string{"OK"}}
	m := NewAppleMusic(r, t.TempDir())
	if err := m.Resume(context.Background(), PlaybackContext{PlaylistID: "PL", TrackID: "T", PositionSec: 61, After: true}); err != nil {
		t.Fatal(err)
	}
	if c := r.calls[0]; len(c.args) < 2 || c.args[0] != "PL" || c.args[1] != "T" || !strings.Contains(c.script, "i + 1") {
		t.Errorf("appel : %+v", c)
	}
}

// Morceau interrompu introuvable dans la playlist : on le reprend où il en était.
func TestAppleMusicResumeAfterFallsBack(t *testing.T) {
	r := &fakeRunner{outputs: []string{"NOTFOUND", "OK"}}
	m := NewAppleMusic(r, t.TempDir())
	m.Resume(context.Background(), PlaybackContext{PlaylistID: "PL", TrackID: "T", PositionSec: 61, After: true})
	if len(r.calls) != 2 || r.calls[1].args[2] != "61" {
		t.Errorf("appels : %+v", r.calls)
	}
}

// Réécoute sans vider la file : le morceau est lancé dans la playlist (repli : la bibliothèque), jamais
// « with once ».
func TestAppleMusicPlayInContext(t *testing.T) {
	r := &fakeRunner{outputs: []string{"OK"}}
	m := NewAppleMusic(r, t.TempDir())
	if err := m.PlayInContext(context.Background(), "PL", "T"); err != nil {
		t.Fatal(err)
	}
	c := r.calls[0]
	if len(c.args) != 2 || c.args[0] != "PL" || c.args[1] != "T" || strings.Contains(c.script, "with once") || !strings.Contains(c.script, "library playlist 1") {
		t.Errorf("appel : %+v", c)
	}
}

// En aléatoire, la reprise relance la playlist entière : c'est la seule commande qui remplit la file de
// Musique (un « play » sur un morceau précis la laisse vide).
func TestAppleMusicResumeAfterShuffleRefillsQueue(t *testing.T) {
	r := &fakeRunner{outputs: []string{"OK"}}
	m := NewAppleMusic(r, t.TempDir())
	m.Resume(context.Background(), PlaybackContext{PlaylistID: "PL", TrackID: "T", After: true})
	if !strings.Contains(r.calls[0].script, "if shuffle enabled then") || !strings.Contains(r.calls[0].script, "play p") {
		t.Errorf("script :\n%s", r.calls[0].script)
	}
}
