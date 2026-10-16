package app

import (
	"context"
	"strings"
	"testing"
	"time"

	"vinyl/internal/catalog"
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
func (f *fakeBack) JumpForward(_ context.Context, steps int, id string, pos int) error {
	f.jumps = append(f.jumps, [3]any{steps, id, pos})
	return nil
}

// Comme un clic dans l'Historique de Musique : la file n'est jamais remplacée. Précédent ×N vers le
// morceau choisi, puis à sa fin, suivant ×(N+1) : le morceau interrompu reste dans l'historique et la
// file reprend au morceau qui devait suivre, depuis son début.
func TestReplayKeepsMusicQueueAndComesBack(t *testing.T) {
	p := &fakeBack{backOK: true}
	p.saved = player.PlaybackContext{PlaylistID: "PL", TrackID: "4", PositionSec: 61}
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil) // pile 3 2 1, en cours 4
	if err := s.PlayFromHistory("applemusic", "2"); err != nil {
		t.Fatal(err)
	}
	if len(p.backs) != 1 || p.backs[0] != [2]any{"2", 2} || len(p.alone) != 0 {
		t.Fatalf("précédent : %v, joué seul : %v", p.backs, p.alone)
	}
	s.ObservePlayback(playing("2", 1000, 200000))
	if len(p.jumps) != 0 {
		t.Fatal("pas de retour avant la fin")
	}
	s.ObservePlayback(playing("2", 199000, 200000)) // dernière seconde
	if len(p.jumps) != 1 || p.jumps[0] != [3]any{3, "", 0} {
		t.Fatalf("retour : %v", p.jumps)
	}
	s.ObservePlayback(playing("4", 61500, 300000))
	if len(p.jumps) != 1 {
		t.Error("un seul retour")
	}
}

// Fin manquée (Musique est déjà passé au morceau suivant) : un « suivant » de moins.
func TestReplayComesBackEvenIfTheEndWasMissed(t *testing.T) {
	p := &fakeBack{backOK: true}
	p.saved = player.PlaybackContext{PlaylistID: "PL", TrackID: "4", PositionSec: 10}
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil)
	s.PlayFromHistory("applemusic", "1") // 3 morceaux en arrière
	s.ObservePlayback(playing("1", 197000, 200000))
	s.ObservePlayback(playing("2", 300, 200000))
	if len(p.jumps) != 1 || p.jumps[0] != [3]any{3, "", 0} {
		t.Fatalf("retour : %v", p.jumps)
	}
}

// Morceau introuvable dans la file de Musique : surtout pas de nouveau « play » (la file serait
// vidée) ; une erreur explique pourquoi rien ne se passe.
func TestReplayNeverReplacesMusicQueue(t *testing.T) {
	p := &fakeBack{backOK: false}
	p.saved = player.PlaybackContext{PlaylistID: "PL", TrackID: "4", PositionSec: 61}
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil)
	if err := s.PlayFromHistory("applemusic", "2"); err == nil {
		t.Error("erreur attendue (morceau plus dans la file)")
	}
	if len(p.alone) != 0 || len(p.played) != 0 {
		t.Errorf("rien ne doit être lancé : seul %v, lancé %v", p.alone, p.played)
	}
}

// Pile décalée : le morceau est trouvé un cran plus loin ; le retour utilise le vrai nombre de pas.
func TestReplayUsesTheStepsReallyTaken(t *testing.T) {
	p := &fakeBack{backOK: true, extra: 1}
	p.saved = player.PlaybackContext{PlaylistID: "PL", TrackID: "4", PositionSec: 5}
	s := serviceWith(p)
	s.SetHistory(pileOf(t, "1", "2", "3", "4"), nil)
	s.PlayFromHistory("applemusic", "2")
	s.ObservePlayback(playing("2", 1000, 200000))
	s.ObservePlayback(playing("2", 199000, 200000))
	if len(p.jumps) != 1 || p.jumps[0] != [3]any{4, "", 0} {
		t.Errorf("retour : %v", p.jumps)
	}
}

func TestReplayBackCancelledWhenTheUserPicksSomethingElse(t *testing.T) {
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
