// Package app expose le backend au frontend Wails.
package app

import (
	"context"
	"encoding/base64"
	"errors"
	"math/rand/v2"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"vinyl/internal/auth"
	"vinyl/internal/catalog"
	"vinyl/internal/history"
	"vinyl/internal/lyrics"
	"vinyl/internal/player"
	"vinyl/internal/settings"
	"vinyl/internal/stats"
	"vinyl/internal/sysvolume"
	"vinyl/internal/widgetupdate"
)

const callLimit = 3 * time.Second

type VinylService struct {
	store       *settings.Store
	watcher     *player.Watcher
	catalogs    map[string]catalog.Catalog
	music       *catalog.MusicLibrary
	spotifyAuth *auth.SpotifyAuth
	widget      *widgetupdate.Manager
	stats       *stats.Store
	history     *history.Store
	onHistory   func([]history.Entry) // diffusion de la pile au frontend
	openLink    func(string) error    // ouverture d'un lien (remplacée dans les tests)
	// spotifyPlay : lecture Spotify par l'API web (hors Mac) ; nil sans catalogue Spotify.
	spotifyPlay func(ctx context.Context, contextURI, trackURI string, shuffle bool) error
	queueMu     sync.Mutex
	queue       *vinylQueue // morceaux enchaînés par Vinyl (album, réécoute depuis la pile)
	back        *backReplay // réécoute par « précédent » dans la file du lecteur (Windows)
	version     string      // numéro de la release, « dev » pour une fabrication à la main
	upd         Updater     // système de mise à jour (nil en « dev »)
	exePath     func() (string, error)
	spec        *spectrum // capture du son pour l'égaliseur (remplacée dans les tests)
	lyricsFind  func(ctx context.Context, title, artist, album string, durationMs int64) (lyrics.Lyrics, error)
	lyricsMu    sync.Mutex
	lyricsCache map[string]lyrics.Lyrics // par source et morceau (la dernière trentaine)
}

func New(store *settings.Store, watcher *player.Watcher, music *catalog.MusicLibrary) *VinylService {
	s := &VinylService{store: store, watcher: watcher, music: music, catalogs: map[string]catalog.Catalog{}, openLink: openURL, version: "dev", exePath: os.Executable,
		lyricsFind: lyrics.NewLRCLib("dev").Find, lyricsCache: map[string]lyrics.Lyrics{},
		spec: newSpectrum(systemTap{}, func(b []float32) { application.Get().Event.Emit("spectrum", b) })}
	if music != nil { // bibliothèque Apple Music : Mac seulement
		s.catalogs["applemusic"] = music
	}
	return s
}

// SetCatalog enregistre le catalogue d'une source (utilisé pour Spotify).
func (s *VinylService) SetCatalog(source string, c catalog.Catalog) {
	s.catalogs[source] = c
	if w, ok := c.(*catalog.SpotifyWeb); ok && source == "spotify" {
		s.spotifyPlay = w.Play
	}
}

// SetVersion : numéro de la release (sans « v »), « dev » pour une fabrication à la main.
func (s *VinylService) SetVersion(v string) {
	if v != "" {
		s.version = v
		s.lyricsFind = lyrics.NewLRCLib(v).Find
	}
}

// Version : numéro de version de l'app (affiché dans Réglages).
func (s *VinylService) Version() string { return s.version }

var errNoPlayer = errors.New("aucun lecteur actif")

func (s *VinylService) withActive(f func(ctx context.Context, p player.Player) error) error {
	p := s.watcher.Active()
	if p == nil {
		return errNoPlayer
	}
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	err := f(ctx, p)
	s.watcher.Wake()
	return err
}

func (s *VinylService) GetSnapshot() player.Snapshot { return s.watcher.Current() }

func (s *VinylService) PlayPause() error {
	return s.withActive(func(ctx context.Context, p player.Player) error { return p.PlayPause(ctx) })
}

func (s *VinylService) Next() error {
	if q := s.activeQueue(); q != nil {
		return s.step(q, +1, false)
	}
	return s.withActive(func(ctx context.Context, p player.Player) error { return p.Next(ctx) })
}

func (s *VinylService) Previous() error {
	if q := s.activeQueue(); q != nil {
		// Comme un lecteur : au-delà de 3 s, retour au début du morceau ; sinon, morceau précédent.
		if s.watcher.Current().NowPlaying.PositionMs > 3000 || q.idx == 0 {
			return s.withActive(func(ctx context.Context, p player.Player) error { return p.Seek(ctx, 0) })
		}
		return s.step(q, -1, false)
	}
	return s.withActive(func(ctx context.Context, p player.Player) error { return p.Previous(ctx) })
}

func (s *VinylService) SeekMs(ms int64) error {
	return s.withActive(func(ctx context.Context, p player.Player) error {
		return p.Seek(ctx, time.Duration(ms)*time.Millisecond)
	})
}

// PlayItem lance une playlist, un album ou un titre sur la source donnée ("applemusic" ou "spotify").
func (s *VinylService) PlayItem(source, id string) error {
	p := s.watcher.Player(source)
	if p == nil {
		return errors.New("source inconnue")
	}
	if _, ok := p.(player.ReplayResumer); ok && strings.HasPrefix(id, "album:") {
		return s.PlayFrom(source, id, "", false) // album : joué titre par titre par Vinyl
	}
	if _, mac := p.(*player.Spotify); source == "spotify" && !mac && s.spotifyPlay != nil {
		return s.playSpotifyWeb(id) // hors Mac : API web, repli sur le lien
	}
	s.setQueue(nil) // l'utilisateur lance autre chose : Vinyl ne pilote plus la suite
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	err := p.PlayItem(ctx, id)
	s.watcher.Wake()
	return err
}

func (s *VinylService) GetSettings() settings.Settings { return s.store.Get() }

// UpdateSettings applique une mise à jour partielle (seuls les champs fournis changent) et la diffuse.
func (s *VinylService) UpdateSettings(p settings.Patch) (settings.Settings, error) {
	v, err := s.store.Update(p)
	if err != nil {
		return v, err
	}
	application.Get().Event.Emit("settings", v)
	s.watcher.Wake()
	return v, nil
}

func (s *VinylService) OpenMainWindow() {
	if w, ok := application.Get().Window.GetByName("main"); ok {
		w.Show()
		w.Focus()
	}
}

// OpenAutomationSettings ouvre Réglages Système > Confidentialité et sécurité > Automatisation.
func (s *VinylService) OpenAutomationSettings() error {
	if runtime.GOOS != "darwin" {
		return nil // réglage propre au Mac
	}
	return exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation").Run()
}

// LaunchAtLogin : Vinyl s'ouvre à l'ouverture de session (état tenu par le système : éléments
// d'ouverture du Mac, clé Run de Windows).
func (s *VinylService) LaunchAtLogin() bool {
	on, _ := application.Get().Autostart.IsEnabled()
	return on
}

// SetLaunchAtLogin active ou retire le lancement au démarrage et renvoie l'état obtenu.
func (s *VinylService) SetLaunchAtLogin(on bool) (bool, error) {
	a := application.Get().Autostart
	var err error
	if on {
		err = a.Enable()
	} else {
		err = a.Disable()
	}
	return s.LaunchAtLogin(), err
}

func (s *VinylService) catalog(source string) (catalog.Catalog, error) {
	c, ok := s.catalogs[source]
	if !ok {
		return nil, catalog.ErrUnsupported
	}
	return c, nil
}

func (s *VinylService) Library(source, kind string, offset int) (catalog.Page, error) {
	c, err := s.catalog(source)
	if err != nil {
		return catalog.Page{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return c.Library(ctx, kind, offset)
}

func (s *VinylService) Artist(source, name, trackID string) (catalog.ArtistPage, error) {
	c, err := s.catalog(source)
	if err != nil {
		return catalog.ArtistPage{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	return c.Artist(ctx, name, trackID)
}

// ItemArtwork renvoie la pochette d'un élément de la bibliothèque Musique (chargée à la demande).
func (s *VinylService) ItemArtwork(source, id string) (string, error) {
	if source != "applemusic" || s.music == nil {
		return "", nil
	}
	// Délai large : la demande peut attendre son tour (2 exports à la fois).
	// Miniature légère pour l'étagère : bien plus rapide à décoder pendant le défilement.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return s.music.Thumbnail(ctx, id)
}

// PlayCatalogTrack joue un titre de page artiste : bibliothèque si possible, sinon ouverture dans l'app.
func (s *VinylService) PlayCatalogTrack(source string, item catalog.Item, artist string) error {
	if source == "spotify" {
		return s.PlayItem("spotify", item.ID)
	}
	if am, ok := s.watcher.Player("applemusic").(*player.AppleMusic); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if found, err := am.PlayLibraryTrack(ctx, item.Title, artist); err == nil && found {
			s.watcher.Wake()
			return nil
		}
	}
	return s.OpenURL(item.URL)
}

// OpenURL ouvre un lien Apple Music dans l'app Musique (https → music://), sinon dans le navigateur.
func (s *VinylService) OpenURL(u string) error {
	if !strings.HasPrefix(u, "music://") && !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "spotify:") {
		return errors.New("lien refusé")
	}
	return openURL(u)
}

// Platform : « darwin » ou « windows » ; le frontend masque ce qui n'existe pas sur le système.
func (s *VinylService) Platform() string { return runtime.GOOS }

func (s *VinylService) SetSpotifyAuth(a *auth.SpotifyAuth) { s.spotifyAuth = a }

func (s *VinylService) SpotifyLogin() error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	return s.spotifyAuth.Login(ctx)
}

func (s *VinylService) SpotifyLogout() error { return s.spotifyAuth.Logout() }

func (s *VinylService) SpotifyStatus() bool { return s.spotifyAuth.LoggedIn() }

// SaveShareImage demande où enregistrer la carte de partage puis écrit le PNG. Renvoie "" si annulé.
func (s *VinylService) SaveShareImage(pngBase64, suggestedName string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(pngBase64)
	if err != nil {
		return "", err
	}
	path, err := application.Get().Dialog.SaveFile().
		SetFilename(suggestedName).
		AddFilter("Image PNG", "*.png").
		CanCreateDirectories(true).
		PromptForSingleSelection()
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o644)
}

// SetWidgetUpdates branche le gestionnaire de mise à jour du widget de bureau.
func (s *VinylService) SetWidgetUpdates(m *widgetupdate.Manager) { s.widget = m }

// WidgetUpdateStatus indique si l'icône ☁️↓ doit s'afficher.
func (s *VinylService) WidgetUpdateStatus() widgetupdate.Status {
	if s.widget == nil {
		return widgetupdate.Status{State: widgetupdate.StateNone}
	}
	return s.widget.Status()
}

// UpdateWidget lance la recompilation et la réinstallation du widget en arrière-plan.
func (s *VinylService) UpdateWidget() error {
	if s.widget == nil {
		return errors.New("mise à jour du widget indisponible")
	}
	return s.widget.Start()
}

func (s *VinylService) SetStats(st *stats.Store) { s.stats = st }

// SetHistory branche la pile des derniers morceaux écoutés et sa diffusion (notify peut être nil).
func (s *VinylService) SetHistory(h *history.Store, notify func([]history.Entry)) {
	s.history, s.onHistory = h, notify
}

// PlayFromHistory réécoute un morceau de la pile sans vider la file du lecteur : le morceau en cours est
// sauté, le morceau choisi est joué, puis la file continue normalement (au morceau qui suivait celui
// qu'on a sauté). La pile n'est pas touchée.
func (s *VinylService) PlayFromHistory(source, trackID string) error {
	p := s.watcher.Player(source)
	if p == nil {
		return errors.New("source inconnue")
	}
	// Album enchaîné par Vinyl : c'est sa propre file, le morceau y est glissé à la place du morceau en cours.
	if r, ok := p.(player.ReplayResumer); ok {
		if q := s.activeQueue(); q != nil && q.source == source {
			return s.replaceCurrent(r, q, trackID)
		}
	}
	// Spotify : le morceau est mis en tête de la file, puis « suivant ».
	if source == "spotify" {
		web, ok := s.catalogs["spotify"].(*catalog.SpotifyWeb)
		if !ok || s.spotifyAuth == nil || !s.spotifyAuth.LoggedIn() {
			return errors.New("connecte Spotify dans les réglages pour réécouter un morceau de la pile")
		}
		ctx, cancel := context.WithTimeout(context.Background(), callLimit)
		defer cancel()
		defer s.watcher.Wake()
		return web.PlayNext(ctx, trackID)
	}
	// Musique : « Lire ensuite » puis « suivant » (voir AppleMusic.PlayNext).
	if np, ok := p.(player.NextPlayer); ok {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		defer s.watcher.Wake()
		return np.PlayNext(ctx, trackID)
	}
	// Lecteur qui ne sait que reculer / avancer dans sa file (Windows) : « précédent » jusqu'au morceau
	// (son coupé, morceaux traversés ni affichés ni posés sur la pile) ; à sa fin, « suivant » jusqu'au
	// morceau qui suivait celui qu'on a sauté (voir observeBack).
	b, back := p.(player.BackReplayer)
	cs, saves := p.(player.ContextSaver)
	if back && saves {
		if steps := s.stepsBack(source, trackID); steps > 0 {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // un pas peut prendre 2,5 s
			defer cancel()
			saved, err := cs.SaveContext(ctx)
			if err == nil && saved.TrackID != "" {
				s.watcher.Hold(true)
				taken, err := b.ReplayBack(ctx, trackID, steps, saved.PositionSec)
				s.watcher.Hold(false)
				s.watcher.Wake()
				if err != nil {
					return err
				}
				if taken > 0 {
					s.queueMu.Lock()
					s.back = &backReplay{source: source, trackID: trackID, steps: taken, saved: saved, started: time.Now()}
					s.queueMu.Unlock()
					return nil
				}
			}
		}
	}
	return errors.New("ce morceau n'est plus dans la file de lecture")
}

// CurrentPlaylist : playlist en cours de lecture, au format des éléments de la bibliothèque
// (« playlist:<id> » pour Apple Music, URI pour Spotify) ; "" si inconnue.
func (s *VinylService) CurrentPlaylist(source string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	switch source {
	case "applemusic":
		cs, ok := s.watcher.Player("applemusic").(player.ContextSaver)
		if !ok {
			return "", nil
		}
		c, err := cs.SaveContext(ctx)
		if err != nil || c.PlaylistID == "" {
			return "", nil
		}
		return "playlist:" + c.PlaylistID, nil
	case "spotify":
		if w, ok := s.catalogs["spotify"].(*catalog.SpotifyWeb); ok {
			if id, err := w.CurrentPlaylist(ctx); err == nil {
				return id, nil
			}
		}
	}
	return "", nil
}

// replaceCurrent : dans la file de Vinyl, le morceau choisi prend la place du morceau en cours, la suite
// reste la même.
func (s *VinylService) replaceCurrent(r player.ReplayResumer, prev *vinylQueue, trackID string) error {
	s.queueMu.Lock()
	q := &vinylQueue{source: prev.source, tracks: append([]player.Track{{ID: trackID}}, prev.tracks[prev.idx+1:]...)}
	s.queueMu.Unlock()
	return s.startQueue(r, q)
}

// vinylQueue : morceaux que Vinyl enchaîne lui-même, chacun joué « une fois » dans Musique (aucune
// playlist créée) : un album.
type vinylQueue struct {
	source       string
	tracks       []player.Track
	idx          int
	started      time.Time
	seen         bool  // le morceau en cours a bien démarré
	lastPos, dur int64 // dernière position vue (ms) et durée
}

// nearEnd : le morceau était dans ses dernières secondes au dernier sondage.
func (q *vinylQueue) nearEnd() bool { return q.seen && q.dur > 0 && q.lastPos >= q.dur-4000 }

func (s *VinylService) setQueue(q *vinylQueue) {
	s.queueMu.Lock()
	s.queue = q
	s.back = nil
	s.queueMu.Unlock()
}

// backReplay : morceau réécouté en reculant dans la file de Musique ; à sa fin, comme après un clic dans
// l'Historique de Musique, on avance au morceau qui suivait le morceau interrompu (steps+1 « suivant ») :
// le morceau interrompu reste dans l'historique et la file continue normalement.
type backReplay struct {
	source, trackID string
	steps           int
	saved           player.PlaybackContext
	started         time.Time
	seen            bool
	lastPos, dur    int64
}

// stepsBack : à combien de morceaux en arrière se trouve trackID (1 = celui d'avant le morceau en
// cours), d'après la pile ; 0 s'il n'y est pas.
func (s *VinylService) stepsBack(source, trackID string) int {
	for i, e := range s.History() {
		if e.Source == source && e.TrackID == trackID {
			return i + 1
		}
	}
	return 0
}

// observeBack suit une réécoute par « précédent » ; renvoie true si elle est en cours.
func (s *VinylService) observeBack(np player.NowPlaying) bool {
	s.queueMu.Lock()
	b := s.back
	if b == nil {
		s.queueMu.Unlock()
		return false
	}
	steps := -1 // < 0 : rien à faire
	drop := false
	switch {
	case np.Source != "" && np.Source != b.source, np.State == player.StateStopped:
		drop = b.seen
	case np.TrackID == b.trackID:
		b.seen, b.lastPos, b.dur = true, np.PositionMs, np.DurationMs
		if b.dur > 0 && np.PositionMs >= b.dur-1500 {
			steps = b.steps + 1 // dernière seconde : directement au morceau qui suivait l'interrompu
		}
	case !b.seen:
		drop = time.Since(b.started) > 15*time.Second
	case b.dur > 0 && b.lastPos >= b.dur-4000:
		steps = b.steps // Musique est déjà passé au morceau suivant
	default:
		drop = true // l'utilisateur a choisi autre chose
	}
	if drop || steps >= 0 {
		s.back = nil
	}
	s.queueMu.Unlock()
	if steps >= 0 {
		if j, ok := s.watcher.Player(b.source).(player.BackReplayer); ok {
			ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
			defer cancel()
			j.JumpForward(ctx, steps, "", 0)
			s.watcher.Wake()
		}
	}
	return true
}

// activeQueue renvoie les morceaux enchaînés par Vinyl s'ils concernent le lecteur actif.
func (s *VinylService) activeQueue() *vinylQueue {
	s.queueMu.Lock()
	defer s.queueMu.Unlock()
	if s.queue == nil {
		return nil
	}
	if a := s.watcher.Active(); a != nil && a.Name() != s.queue.source {
		return nil
	}
	return s.queue
}

// startQueue lance le morceau courant de q.
func (s *VinylService) startQueue(r player.ReplayResumer, q *vinylQueue) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	s.queueMu.Lock()
	q.started, q.seen, q.lastPos, q.dur = time.Now(), false, 0, 0
	id := q.tracks[q.idx].ID
	s.queueMu.Unlock()
	if err := r.PlayAlone(ctx, id); err != nil {
		return err
	}
	s.setQueue(q)
	s.watcher.Wake()
	return nil
}

// step passe au morceau suivant (+1) ou précédent (-1). natural : fin du morceau (la répétition du
// lecteur s'applique). Après le dernier : bouclage si « tout répéter », sinon fin.
func (s *VinylService) step(q *vinylQueue, delta int, natural bool) error {
	p := s.watcher.Player(q.source)
	r, ok := p.(player.ReplayResumer)
	if !ok {
		s.setQueue(nil)
		return nil
	}
	rep := player.RepeatOff
	if m, ok := p.(player.Modes); ok {
		ctx, cancel := context.WithTimeout(context.Background(), callLimit)
		_, rep, _ = m.PlaybackModes(ctx)
		cancel()
	}
	s.queueMu.Lock()
	next := q.idx + delta
	if natural && rep == player.RepeatOne {
		next = q.idx
	}
	if next < 0 {
		next = 0
	}
	if next >= len(q.tracks) {
		if rep != player.RepeatAll {
			s.queue = nil
			s.queueMu.Unlock()
			return nil
		}
		next = 0
	}
	q.idx = next
	s.queueMu.Unlock()
	return s.startQueue(r, q)
}

// ObservePlayback est appelé à chaque sondage du lecteur. Quand le morceau joué par Vinyl se termine
// (arrêt, morceau suivant ou répétition), la suite est lancée. Si l'utilisateur choisit lui-même autre
// chose, Vinyl arrête de piloter la suite.
func (s *VinylService) ObservePlayback(snap player.Snapshot) {
	if snap.Problem == "" && s.observeBack(snap.NowPlaying) {
		return
	}
	s.queueMu.Lock()
	q := s.queue
	if q == nil || snap.Problem != "" {
		s.queueMu.Unlock()
		return
	}
	np := snap.NowPlaying
	cur := q.tracks[q.idx].ID
	advance, drop := false, false
	switch {
	case np.Source != "" && np.Source != q.source:
		drop = q.seen
	case np.TrackID == cur:
		if q.nearEnd() && np.PositionMs < 3000 {
			advance = true // revenu au début : fini puis relancé
		} else {
			q.seen, q.lastPos, q.dur = true, np.PositionMs, np.DurationMs
		}
	case !q.seen:
		drop = time.Since(q.started) > 15*time.Second // n'a jamais démarré
	default:
		advance, drop = q.nearEnd(), !q.nearEnd()
	}
	if drop {
		s.queue = nil
	}
	s.queueMu.Unlock()
	if advance {
		s.step(q, +1, true)
	}
}

// History renvoie les derniers morceaux écoutés (le plus récent en premier), pour la pile de la platine.
func (s *VinylService) History() []history.Entry {
	if s.history == nil {
		return []history.Entry{}
	}
	return s.history.List()
}

// Stats renvoie le compteur « Morceaux lus depuis ».
func (s *VinylService) Stats() stats.Snapshot {
	if s.stats == nil {
		return stats.Snapshot{}
	}
	return s.stats.Get()
}

// Tracks renvoie l'album ou la playlist avec ses titres.
func (s *VinylService) Tracks(source, id string) (catalog.Collection, error) {
	c, err := s.catalog(source)
	if err != nil {
		return catalog.Collection{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return c.Tracks(ctx, id)
}

// PlayFrom lance un album ou une playlist à partir d'un titre (vide = début), avec ou sans aléatoire.
// Musique : une playlist de l'utilisateur est jouée telle quelle ; un album est joué titre par titre
// par Vinyl (aucune playlist créée dans la bibliothèque).
func (s *VinylService) PlayFrom(source, collectionID, trackID string, shuffle bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer s.watcher.Wake()
	p := s.watcher.Player(source)
	if sp, ok := p.(*player.Spotify); ok {
		return sp.PlayInContext(ctx, collectionID, trackID, shuffle)
	}
	if source == "spotify" && s.spotifyPlay != nil {
		// Hors Mac : lecture par l'API web ; refusée (pas Premium, aucun appareil) → l'album s'ouvre dans Spotify.
		err := s.spotifyPlay(ctx, collectionID, trackID, shuffle)
		if errors.Is(err, catalog.ErrPlaybackRefused) {
			return s.openLink(collectionID)
		}
		return err
	}
	if pid, ok := strings.CutPrefix(collectionID, "playlist:"); ok {
		if am, ok := p.(*player.AppleMusic); ok {
			s.setQueue(nil)
			return am.PlayInPlaylist(ctx, pid, trackID, shuffle)
		}
	}
	r, ok := p.(player.ReplayResumer)
	if !ok {
		return errors.New("source inconnue")
	}
	col, err := s.Tracks(source, collectionID)
	if err != nil {
		return err
	}
	if len(col.Tracks) == 0 {
		return errors.New("album vide")
	}
	tracks := append([]player.Track{}, col.Tracks...)
	start := 0
	for i, t := range tracks {
		if t.ID == trackID {
			start = i
		}
	}
	if shuffle {
		first := tracks[start]
		rand.Shuffle(len(tracks), func(i, j int) { tracks[i], tracks[j] = tracks[j], tracks[i] })
		for i, t := range tracks { // le titre choisi d'abord
			if t.ID == first.ID {
				tracks[0], tracks[i] = tracks[i], tracks[0]
			}
		}
		start = 0
	}
	return s.startQueue(r, &vinylQueue{source: source, tracks: tracks, idx: start})
}

type QueueState struct {
	Shuffle  bool           `json:"shuffle"`
	Repeat   string         `json:"repeat"`
	Upcoming []player.Track `json:"upcoming"`
	Note     string         `json:"note"`
}

// sourceOf renvoie le nom du lecteur (« applemusic », « spotify »), ou "" s'il n'en a pas.
func sourceOf(m any) string {
	if p, ok := m.(player.Player); ok {
		return p.Name()
	}
	return ""
}

func (s *VinylService) modes() (player.Modes, error) {
	p := s.watcher.Active()
	if p == nil {
		return nil, errNoPlayer
	}
	m, ok := p.(player.Modes)
	if !ok {
		return nil, errors.New("modes de lecture non pris en charge")
	}
	return m, nil
}

// ModesState : aléatoire et répétition du lecteur actif (boutons de la platine).
type ModesState struct {
	Shuffle bool   `json:"shuffle"`
	Repeat  string `json:"repeat"`
}

// Modes renvoie l'aléatoire et la répétition, sans charger la file d'attente (rapide).
func (s *VinylService) Modes() (ModesState, error) {
	m, err := s.modes()
	if err != nil {
		return ModesState{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	sh, rep, err := m.PlaybackModes(ctx)
	if err != nil {
		return ModesState{}, err
	}
	return ModesState{Shuffle: sh, Repeat: string(rep)}, nil
}

// Queue renvoie l'aléatoire, la répétition et les titres à venir du lecteur actif.
func (s *VinylService) Queue() (QueueState, error) {
	m, err := s.modes()
	if err != nil {
		return QueueState{Upcoming: []player.Track{}}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	sh, rep, err := m.PlaybackModes(ctx)
	if err != nil {
		return QueueState{Upcoming: []player.Track{}}, err
	}
	q := QueueState{Shuffle: sh, Repeat: string(rep), Upcoming: []player.Track{}}
	// Album enchaîné par Vinyl : la suite est la sienne.
	if vq := s.activeQueue(); vq != nil {
		s.queueMu.Lock()
		q.Upcoming = append(q.Upcoming, vq.tracks[vq.idx+1:]...)
		s.queueMu.Unlock()
		return q, nil
	}
	if _, mac := m.(*player.Spotify); !mac && sourceOf(m) == "spotify" {
		s.spotifyQueue(ctx, &q) // hors Mac : même file que sur Mac, par l'API web
		return q, nil
	}
	switch p := m.(type) {
	case *player.AppleMusic:
		if sh {
			q.Note = "Ordre aléatoire"
			break
		}
		if ts, err := p.Upcoming(ctx, 30, rep == player.RepeatAll); err == nil && ts != nil {
			q.Upcoming = ts
		}
	case *player.Spotify:
		s.spotifyQueue(ctx, &q)
	default:
		q.Note = "File indisponible" // Apple Music sous Windows
	}
	return q, nil
}

// spotifyQueue : file d'attente Spotify par l'API web.
func (s *VinylService) spotifyQueue(ctx context.Context, q *QueueState) {
	web, ok := s.catalogs["spotify"].(*catalog.SpotifyWeb)
	if !ok || s.spotifyAuth == nil || !s.spotifyAuth.LoggedIn() {
		q.Note = "Connecte Spotify pour voir la suite"
		return
	}
	if ts, err := web.Queue(ctx); err == nil {
		q.Upcoming = ts
	} else {
		q.Note = "File Spotify indisponible"
	}
}

// playSpotifyWeb lance un titre, un album ou une playlist Spotify par l'API web (hors Mac) ; refusé
// (pas Premium, aucun appareil actif) → le lien s'ouvre dans Spotify.
func (s *VinylService) playSpotifyWeb(id string) error {
	if !strings.HasPrefix(id, "spotify:") {
		return errors.New("lien Spotify invalide") // jamais d'autre identifiant vers l'API ni ouvert comme lien
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	defer s.watcher.Wake()
	var err error
	if strings.HasPrefix(id, "spotify:track:") {
		web, _ := s.catalogs["spotify"].(*catalog.SpotifyWeb)
		if web == nil {
			return s.openLink(id)
		}
		err = web.PlayTrack(ctx, id)
	} else {
		err = s.spotifyPlay(ctx, id, "", false)
	}
	if errors.Is(err, catalog.ErrPlaybackRefused) {
		return s.openLink(id)
	}
	return err
}

func (s *VinylService) SetShuffle(on bool) error {
	m, err := s.modes()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	defer s.watcher.Wake()
	return m.SetShuffle(ctx, on)
}

func (s *VinylService) SetRepeat(mode string) error {
	m, err := s.modes()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	return m.SetRepeat(ctx, player.RepeatMode(mode))
}

// ToggleMiniPlatine affiche ou masque la mini-platine flottante (comme ⌘⇧M).
func (s *VinylService) ToggleMiniPlatine() {
	w, ok := application.Get().Window.GetByName("widget")
	if !ok {
		return
	}
	if w.IsVisible() {
		w.Hide()
	} else {
		w.Show()
	}
}

// PollInterval : fréquence de sondage du lecteur, pour consommer le moins possible. La position est
// interpolée par la page entre deux sondages ; les actions de l'utilisateur réveillent le sondage.
func (s *VinylService) PollInterval() time.Duration {
	s.queueMu.Lock()
	tracked := s.back != nil || s.queue != nil
	s.queueMu.Unlock()
	return pollIntervalFor(s.watcher.Current().NowPlaying, tracked)
}

// pollIntervalFor : 250 ms dans les dernières secondes d'un morceau que Vinyl doit enchaîner (réécoute
// depuis la pile, album), 1 s en lecture, 3 s sinon.
func pollIntervalFor(np player.NowPlaying, tracked bool) time.Duration {
	if np.State != player.StatePlaying {
		return 3 * time.Second
	}
	if tracked && np.DurationMs > 0 && np.DurationMs-np.PositionMs < 3500 {
		return 250 * time.Millisecond
	}
	return time.Second
}

// Volume renvoie le volume de sortie de l'ordinateur (0 à 100, 0 si le son est coupé).
func (s *VinylService) Volume() (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	return sysvolume.Get(ctx)
}

// SetVolume règle le volume de sortie de l'ordinateur (0 à 100) et rétablit le son.
func (s *VinylService) SetVolume(level int) error {
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	return sysvolume.Set(ctx, level)
}

// Lyrics renvoie les paroles d'un morceau : synchronisées (LRCLIB) de préférence, sinon celles enregistrées
// dans Musique, sinon le texte simple de LRCLIB. Un résultat (même vide) est gardé pour ce morceau ; une
// erreur réseau ne l'est pas (nouvel essai au prochain affichage).
func (s *VinylService) Lyrics(source, trackID, title, artist, album string, durationMs int64) (lyrics.Lyrics, error) {
	key := source + "\x1f" + trackID
	s.lyricsMu.Lock()
	l, ok := s.lyricsCache[key]
	s.lyricsMu.Unlock()
	if ok {
		return l, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	l, err := s.lyricsFind(ctx, title, artist, album, durationMs)
	if len(l.Synced) == 0 && source == "applemusic" {
		if p, ok := s.watcher.Active().(interface {
			Lyrics(ctx context.Context, trackID string) (string, error)
		}); ok {
			if plain, e := p.Lyrics(ctx, trackID); e == nil && plain != "" {
				l, err = lyrics.Lyrics{Plain: plain}, nil
			}
		}
	}
	if err != nil {
		return lyrics.Lyrics{}, err
	}
	s.lyricsMu.Lock()
	if len(s.lyricsCache) >= 30 {
		clear(s.lyricsCache)
	}
	s.lyricsCache[key] = l
	s.lyricsMu.Unlock()
	return l, nil
}

type favoriter interface {
	Favorite(ctx context.Context, trackID string) (bool, error)
	SetFavorite(ctx context.Context, trackID string, on bool) error
}

type spotifyLikes interface {
	Loved(ctx context.Context, id string) (bool, error)
	SetLoved(ctx context.Context, id string, on bool) error
}

// favorites : Musique par le lecteur actif (AppleScript), Spotify par l'API web (titres likés).
func (s *VinylService) favorites(source string) (favoriter, error) {
	switch source {
	case "applemusic":
		if p, ok := s.watcher.Active().(favoriter); ok {
			return p, nil
		}
	case "spotify":
		if c, ok := s.catalogs[source].(spotifyLikes); ok {
			return likesFavoriter{c}, nil
		}
	}
	return nil, catalog.ErrUnsupported
}

type likesFavoriter struct{ c spotifyLikes }

func (l likesFavoriter) Favorite(ctx context.Context, id string) (bool, error) {
	return l.c.Loved(ctx, id)
}
func (l likesFavoriter) SetFavorite(ctx context.Context, id string, on bool) error {
	return l.c.SetLoved(ctx, id, on)
}

// IsFavorite dit si le morceau est en favori (cœur de Musique, titre liké sur Spotify).
func (s *VinylService) IsFavorite(source, trackID string) (bool, error) {
	f, err := s.favorites(source)
	if err != nil {
		return false, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	return f.Favorite(ctx, trackID)
}

// SetFavorite met le morceau en favori, ou l'en retire.
func (s *VinylService) SetFavorite(source, trackID string, on bool) error {
	f, err := s.favorites(source)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	return f.SetFavorite(ctx, trackID, on)
}

// EditablePlaylists : playlists où l'on peut ajouter le morceau en cours (les siennes, pas les
// intelligentes ; Spotify : les siennes et les collaboratives), triées par nom.
func (s *VinylService) EditablePlaylists(source string) ([]catalog.Item, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	var items []catalog.Item
	switch source {
	case "applemusic":
		am, ok := s.watcher.Player(source).(interface {
			EditablePlaylists(ctx context.Context) ([]player.PlaylistRef, error)
		})
		if !ok {
			return nil, catalog.ErrUnsupported
		}
		ps, err := am.EditablePlaylists(ctx)
		if err != nil {
			return nil, err
		}
		for _, p := range ps {
			if !slices.Contains(catalog.OldVinylPlaylists, p.Name) {
				items = append(items, catalog.Item{ID: p.ID, Kind: "playlist", Title: p.Name})
			}
		}
	case "spotify":
		web, ok := s.catalogs["spotify"].(*catalog.SpotifyWeb)
		if !ok || s.spotifyAuth == nil || !s.spotifyAuth.LoggedIn() {
			return nil, errors.New("connecte Spotify dans les réglages pour ajouter à une playlist")
		}
		var err error
		if items, err = web.EditablePlaylists(ctx); err != nil {
			return nil, err
		}
	default:
		return nil, catalog.ErrUnsupported
	}
	sort.SliceStable(items, func(i, j int) bool { return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title) })
	return items, nil
}

// AddToPlaylist ajoute le morceau à la playlist ; false s'il y était déjà (Musique).
func (s *VinylService) AddToPlaylist(source, trackID, playlistID string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	switch source {
	case "applemusic":
		am, ok := s.watcher.Player(source).(interface {
			AddToPlaylist(ctx context.Context, trackID, playlistID string) (bool, error)
		})
		if !ok {
			return false, catalog.ErrUnsupported
		}
		return am.AddToPlaylist(ctx, trackID, playlistID)
	case "spotify":
		web, ok := s.catalogs["spotify"].(*catalog.SpotifyWeb)
		if !ok || s.spotifyAuth == nil || !s.spotifyAuth.LoggedIn() {
			return false, errors.New("connecte Spotify dans les réglages pour ajouter à une playlist")
		}
		return true, web.AddToPlaylist(ctx, playlistID, trackID)
	}
	return false, catalog.ErrUnsupported
}
