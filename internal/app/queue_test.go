package app

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"vinyl/internal/catalog"
	"vinyl/internal/history"
	"vinyl/internal/player"
)

// fakeCatalog : un album de trois titres A, B, C.
type fakeCatalog struct{}

func (fakeCatalog) Library(context.Context, string, int) (catalog.Page, error) {
	return catalog.Page{}, nil
}
func (fakeCatalog) Artist(context.Context, string, string) (catalog.ArtistPage, error) {
	return catalog.ArtistPage{}, nil
}
func (fakeCatalog) Tracks(context.Context, string) (catalog.Collection, error) {
	return catalog.Collection{Tracks: []player.Track{{ID: "A", Title: "a"}, {ID: "B", Title: "b"}, {ID: "C", Title: "c"}}}, nil
}

func albumService(p *fakeResumer) *VinylService {
	s := serviceWith(p)
	s.SetCatalog("applemusic", fakeCatalog{})
	return s
}

func stopped() player.Snapshot {
	return player.Snapshot{NowPlaying: player.NowPlaying{Source: "applemusic", State: player.StateStopped}}
}

// Fin naturelle du morceau id (Musique s'arrête : il était joué « une fois »).
func finish(s *VinylService, id string) {
	s.ObservePlayback(playing(id, 1000, 200000))
	s.ObservePlayback(playing(id, 199000, 200000))
	s.ObservePlayback(stopped())
}

// Sans créer de playlist dans Musique : Vinyl joue l'album titre par titre, dans l'ordre.
func TestAlbumPlaysTrackByTrackWithoutPlaylist(t *testing.T) {
	p := &fakeResumer{}
	s := albumService(p)
	if err := s.PlayFrom("applemusic", "album:X", "B", false); err != nil {
		t.Fatal(err)
	}
	if q, _ := s.Queue(); len(q.Upcoming) != 1 || q.Upcoming[0].ID != "C" {
		t.Errorf("file = %+v", q.Upcoming)
	}
	finish(s, "B")
	finish(s, "C")
	if got := p.alone; len(got) != 2 || got[0] != "B" || got[1] != "C" {
		t.Errorf("joués : %v", got)
	}
	s.ObservePlayback(stopped())
	if len(p.alone) != 2 {
		t.Error("fin de l'album : plus rien")
	}
}

func TestAlbumRepeatAllLoops(t *testing.T) {
	p := &fakeResumer{}
	p.repeat = player.RepeatAll
	s := albumService(p)
	s.PlayFrom("applemusic", "album:X", "C", false)
	finish(s, "C")
	if got := p.alone; len(got) != 2 || got[1] != "A" {
		t.Errorf("joués : %v", got)
	}
}

func TestNextAndPreviousFollowTheAlbum(t *testing.T) {
	p := &fakeResumer{}
	s := albumService(p)
	s.PlayFrom("applemusic", "album:X", "B", false)
	s.ObservePlayback(playing("B", 1000, 200000))
	if err := s.Next(); err != nil {
		t.Fatal(err)
	}
	s.ObservePlayback(playing("C", 500, 200000))
	if err := s.Previous(); err != nil {
		t.Fatal(err)
	}
	if got := p.alone; len(got) != 3 || got[1] != "C" || got[2] != "B" {
		t.Errorf("joués : %v", got)
	}
}

func TestShuffledAlbumPlaysEveryTrackOnce(t *testing.T) {
	p := &fakeResumer{}
	s := albumService(p)
	s.PlayFrom("applemusic", "album:X", "", true)
	for range 3 {
		finish(s, p.alone[len(p.alone)-1])
	}
	seen := map[string]bool{}
	for _, id := range p.alone {
		seen[id] = true
	}
	if len(p.alone) != 3 || len(seen) != 3 {
		t.Errorf("joués : %v", p.alone)
	}
}

func TestPlayItemAlbumUsesTheQueue(t *testing.T) {
	p := &fakeResumer{}
	s := albumService(p)
	if err := s.PlayItem("applemusic", "album:X"); err != nil {
		t.Fatal(err)
	}
	if len(p.alone) != 1 || p.alone[0] != "A" {
		t.Errorf("joués : %v", p.alone)
	}
}

// L'utilisateur choisit autre chose dans Musique : Vinyl arrête de piloter la suite.
func TestAlbumQueueStopsWhenTheUserPicksSomethingElse(t *testing.T) {
	p := &fakeResumer{}
	s := albumService(p)
	s.PlayFrom("applemusic", "album:X", "A", false)
	s.ObservePlayback(playing("A", 30000, 200000))
	s.ObservePlayback(playing("ZZ", 500, 200000))
	s.ObservePlayback(stopped())
	if len(p.alone) != 1 {
		t.Errorf("joués : %v", p.alone)
	}
}

// fakeBack : lecteur qui sait naviguer dans sa propre file (précédent / suivant).
type fakeBack struct {
	fakeResumer
	inCtx  [][2]string
	backOK bool
	extra  int // morceaux en plus trouvés au-delà de la pile (pile décalée)
	backs  [][2]any
	jumps  [][3]any
}

func (f *fakeBack) ReplayBack(_ context.Context, id string, steps, _ int) (int, error) {
	f.backs = append(f.backs, [2]any{id, steps})
	if !f.backOK {
		return 0, nil
	}
	return steps + f.extra, nil
}
func (f *fakeBack) PlayInContext(_ context.Context, pl, id string) error {
	f.inCtx = append(f.inCtx, [2]string{pl, id})
	return f.playErr
}
func (f *fakeBack) JumpForward(_ context.Context, steps int, id string, pos int) error {
	f.jumps = append(f.jumps, [3]any{steps, id, pos})
	return nil
}

// Fréquence de sondage : 1 s en lecture, 3 s à l'arrêt, 250 ms dans les dernières secondes d'un
// morceau réécouté depuis la pile (pour revenir au bon moment).
func TestPollInterval(t *testing.T) {
	if d := pollIntervalFor(player.NowPlaying{State: player.StatePlaying, PositionMs: 1000, DurationMs: 200000}, false); d != time.Second {
		t.Errorf("lecture : %v", d)
	}
	if d := pollIntervalFor(player.NowPlaying{State: player.StatePaused}, false); d != 3*time.Second {
		t.Errorf("pause : %v", d)
	}
	if d := pollIntervalFor(player.NowPlaying{State: player.StatePlaying, PositionMs: 198000, DurationMs: 200000}, true); d != 250*time.Millisecond {
		t.Errorf("fin d'un morceau suivi : %v", d)
	}
	if d := pollIntervalFor(player.NowPlaying{State: player.StatePlaying, PositionMs: 198000, DurationMs: 200000}, false); d != time.Second {
		t.Errorf("fin d'un morceau ordinaire : %v", d)
	}
}

// Sous Windows, lancer un album Spotify passe par l'API web ; refusé → ouverture du lien spotify:.
func TestSpotifyPlayFromFallsBackToOpeningTheLink(t *testing.T) {
	p := &fakeModesPlayer{}
	s := serviceWith(p)
	var opened []string
	s.openLink = func(u string) error { opened = append(opened, u); return nil }
	s.spotifyPlay = func(context.Context, string, string, bool) error { return catalog.ErrPlaybackRefused }
	if err := s.PlayFrom("spotify", "spotify:album:A", "spotify:track:T", false); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != "spotify:album:A" {
		t.Errorf("ouvert : %v", opened)
	}
}

// Lecteur « spotify » qui n'est pas le lecteur AppleScript (Windows) : la file vient de l'API web.
type namedModes struct {
	fakeModesPlayer
	name string
}

func (n *namedModes) Name() string { return n.name }

func TestQueueForSpotifyWithoutAppleScript(t *testing.T) {
	s := serviceWith(&namedModes{name: "spotify"})
	q, err := s.Queue()
	if err != nil {
		t.Fatal(err)
	}
	if q.Note != "Connecte Spotify pour voir la suite" {
		t.Errorf("note = %q", q.Note)
	}
}

// Lecteur qui sait reculer / avancer dans sa file mais pas jouer « une fois » (Windows) : la réécoute
// depuis la pile passe quand même par précédent / suivant, puis revient au morceau interrompu.
type backOnly struct {
	fakeModesPlayer
	saved player.PlaybackContext
	backs []string
	jumps []string
}

func (b *backOnly) SaveContext(context.Context) (player.PlaybackContext, error) { return b.saved, nil }
func (b *backOnly) ReplayBack(_ context.Context, id string, steps, _ int) (int, error) {
	b.backs = append(b.backs, id)
	return steps, nil
}
func (b *backOnly) JumpForward(_ context.Context, _ int, id string, _ int) error {
	b.jumps = append(b.jumps, id)
	return nil
}

func TestReplayFromPileWithBackOnlyPlayer(t *testing.T) {
	p := &backOnly{saved: player.PlaybackContext{TrackID: "4", PositionSec: 12}}
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil)
	if err := s.PlayFromHistory("applemusic", "2"); err != nil {
		t.Fatal(err)
	}
	if len(p.backs) != 1 || p.backs[0] != "2" || len(p.played) != 0 {
		t.Fatalf("recul : %v, lancé seul : %v", p.backs, p.played)
	}
	s.ObservePlayback(playing("2", 1000, 200000))
	s.ObservePlayback(playing("2", 199000, 200000))
	if len(p.jumps) != 1 || p.jumps[0] != "" { // au morceau qui suivait l'interrompu, sans le viser
		t.Errorf("retour : %v", p.jumps)
	}
}

// Windows : morceau impossible à retrouver dans la file → message, jamais d'empreinte envoyée à Spotify.
func TestReplayFallbackOnWindowsNeverSendsFingerprint(t *testing.T) {
	p := &backOnly{saved: player.PlaybackContext{}} // contexte inconnu (publicité, app fermée…)
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3"), nil)
	var opened []string
	s.openLink = func(u string) error { opened = append(opened, u); return nil }
	err := s.PlayFromHistory("applemusic", "w1234")
	if err == nil || !strings.Contains(err.Error(), "plus dans la file") || len(p.played) != 0 || len(opened) != 0 {
		t.Errorf("err=%v lancé=%v ouvert=%v", err, p.played, opened)
	}
}

type spotifyBackOnly struct{ backOnly }

func (*spotifyBackOnly) Name() string { return "spotify" }

// Spotify sous Windows : la pile ne connaît que des empreintes (« w… »), jamais envoyées à l'API web ;
// le morceau est retrouvé par « précédent » comme pour Apple Music.
func TestReplaySpotifyFingerprintOnWindowsGoesBack(t *testing.T) {
	p := &spotifyBackOnly{backOnly{saved: player.PlaybackContext{TrackID: "w4", PositionSec: 12}}}
	s := serviceWith(p)
	h := history.NewStore(t.TempDir() + "/h.json")
	for _, id := range []string{"w1", "w2", "w3", "w4"} {
		h.Playing(history.Entry{Source: "spotify", TrackID: id})
	}
	s.SetHistory(h, nil)
	if err := s.PlayFromHistory("spotify", "w2"); err != nil {
		t.Fatal(err)
	}
	if len(p.backs) != 1 || p.backs[0] != "w2" {
		t.Fatalf("recul : %v", p.backs)
	}
}

// Seuls de vrais liens Spotify partent vers l'API web ou s'ouvrent.
func TestPlaySpotifyWebRejectsNonSpotifyIDs(t *testing.T) {
	s := serviceWith(&namedModes{name: "spotify"})
	var opened []string
	s.openLink = func(u string) error { opened = append(opened, u); return nil }
	s.spotifyPlay = func(context.Context, string, string, bool) error { return catalog.ErrPlaybackRefused }
	if err := s.PlayItem("spotify", "w1234"); err == nil || len(opened) != 0 {
		t.Errorf("err=%v ouvert=%v", err, opened)
	}
}

var errNotInLibrary = errors.New("morceau absent de la bibliothèque")

// Clic sur un disque de la pile (Musique) : « précédent » jusqu'au morceau choisi (la file n'est jamais
// remplacée : ni « play », ni morceau joué seul) ; à sa fin, « suivant » jusqu'au morceau qui suivait
// celui qu'on a sauté.
func TestReplaySkipsTheCurrentSongAndKeepsTheQueue(t *testing.T) {
	p := &fakeBack{backOK: true}
	p.saved = player.PlaybackContext{PlaylistID: "PL", TrackID: "4", PositionSec: 61}
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil) // pile 3 2 1, en cours 4
	if err := s.PlayFromHistory("applemusic", "2"); err != nil {
		t.Fatal(err)
	}
	if len(p.backs) != 1 || p.backs[0] != [2]any{"2", 2} || len(p.alone) != 0 || len(p.inCtx) != 0 || len(p.played) != 0 {
		t.Fatalf("précédent : %v, seul : %v, playlist : %v, lancé : %v", p.backs, p.alone, p.inCtx, p.played)
	}
	s.ObservePlayback(playing("2", 1000, 200000))
	if len(p.jumps) != 0 {
		t.Fatal("rien avant la fin")
	}
	s.ObservePlayback(playing("2", 199000, 200000)) // dernière seconde
	// 2 → 3 → 4 (sauté) → 5 : trois « suivant ».
	if len(p.jumps) != 1 || p.jumps[0] != [3]any{3, "", 0} {
		t.Fatalf("suivant : %v", p.jumps)
	}
	s.ObservePlayback(playing("5", 500, 300000))
	if len(p.jumps) != 1 {
		t.Error("un seul retour")
	}
}

// Fin manquée (Musique est déjà passé au morceau d'après) : un « suivant » de moins.
func TestReplayComesBackEvenIfTheEndWasMissed(t *testing.T) {
	p := &fakeBack{backOK: true}
	p.saved = player.PlaybackContext{PlaylistID: "PL", TrackID: "4", PositionSec: 10}
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil)
	s.PlayFromHistory("applemusic", "1") // 3 morceaux en arrière
	s.ObservePlayback(playing("1", 197000, 200000))
	s.ObservePlayback(playing("2", 300, 200000))
	if len(p.jumps) != 1 || p.jumps[0] != [3]any{3, "", 0} {
		t.Fatalf("suivant : %v", p.jumps)
	}
}

// Morceau introuvable en reculant (aléatoire…) : une erreur, et rien ne remplace la file.
func TestReplayNeverReplacesTheQueue(t *testing.T) {
	p := &fakeBack{backOK: false}
	p.saved = player.PlaybackContext{PlaylistID: "PL", TrackID: "4", PositionSec: 61}
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil)
	if err := s.PlayFromHistory("applemusic", "2"); err == nil || !strings.Contains(err.Error(), "plus dans la file") {
		t.Errorf("err = %v", err)
	}
	if len(p.alone) != 0 || len(p.inCtx) != 0 || len(p.played) != 0 {
		t.Errorf("seul : %v, playlist : %v, lancé : %v", p.alone, p.inCtx, p.played)
	}
}

func TestReplayCancelledWhenTheUserPicksSomethingElse(t *testing.T) {
	p := &fakeBack{backOK: true}
	p.saved = player.PlaybackContext{PlaylistID: "PL", TrackID: "4", PositionSec: 61}
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil)
	s.PlayFromHistory("applemusic", "2")
	s.ObservePlayback(playing("2", 30000, 200000))
	s.ObservePlayback(playing("88", 100, 200000))
	s.ObservePlayback(playing("88", 199000, 200000))
	if len(p.jumps) != 0 {
		t.Errorf("aucun retour attendu : %v", p.jumps)
	}
}

// Album enchaîné par Vinyl : le morceau choisi prend la place du morceau en cours, la suite de l'album
// reste la même.
func TestReplayDuringAnAlbumReplacesTheCurrentSong(t *testing.T) {
	p := &fakeResumer{}
	s := albumService(p)
	if err := s.PlayItem("applemusic", "album:X"); err != nil {
		t.Fatal(err)
	}
	s.ObservePlayback(playing("A", 30000, 200000))
	if err := s.PlayFromHistory("applemusic", "Z"); err != nil {
		t.Fatal(err)
	}
	finish(s, "Z")
	if strings.Join(p.alone, " ") != "A Z B" {
		t.Errorf("joués : %v", p.alone)
	}
}

// Spotify sans connexion à l'API web : rien n'est lancé (la file serait remplacée), un message l'explique.
func TestSpotifyReplayNeedsTheWebAPI(t *testing.T) {
	p := &namedModes{name: "spotify"}
	s := serviceWith(p)
	if err := s.PlayFromHistory("spotify", "spotify:track:1"); err == nil || len(p.played) != 0 {
		t.Errorf("err=%v lancé=%v", err, p.played)
	}
}

// Lecteur qui sait « Lire ensuite » (Musique) : c'est tout ce qui est fait, ni recul ni morceau joué seul.
type fakeNext struct {
	fakeBack
	next []string
}

func (f *fakeNext) PlayNext(_ context.Context, id string) error {
	f.next = append(f.next, id)
	return nil
}

func TestReplayUsesPlayNextWhenAvailable(t *testing.T) {
	p := &fakeNext{}
	p.backOK = true
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil)
	if err := s.PlayFromHistory("applemusic", "2"); err != nil {
		t.Fatal(err)
	}
	if len(p.next) != 1 || p.next[0] != "2" || len(p.backs) != 0 || len(p.alone) != 0 || len(p.played) != 0 {
		t.Errorf("lire ensuite : %v, précédent : %v, seul : %v, lancé : %v", p.next, p.backs, p.alone, p.played)
	}
}
