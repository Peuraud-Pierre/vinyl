// Package app expose le backend au frontend Wails.
package app

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"

	"vinyl/internal/auth"
	"vinyl/internal/catalog"
	"vinyl/internal/history"
	"vinyl/internal/player"
	"vinyl/internal/settings"
	"vinyl/internal/stats"
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
}

func New(store *settings.Store, watcher *player.Watcher, music *catalog.MusicLibrary) *VinylService {
	return &VinylService{store: store, watcher: watcher, music: music,
		catalogs: map[string]catalog.Catalog{"applemusic": music}}
}

// SetCatalog enregistre le catalogue d'une source (utilisé pour Spotify).
func (s *VinylService) SetCatalog(source string, c catalog.Catalog) { s.catalogs[source] = c }

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
	return s.withActive(func(ctx context.Context, p player.Player) error { return p.Next(ctx) })
}

func (s *VinylService) Previous() error {
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
	return exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation").Run()
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
	if source != "applemusic" {
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
	if strings.HasPrefix(u, "https://music.apple.com/") {
		u = "music://" + strings.TrimPrefix(u, "https://")
	}
	if !strings.HasPrefix(u, "music://") && !strings.HasPrefix(u, "https://") && !strings.HasPrefix(u, "spotify:") {
		return errors.New("lien refusé")
	}
	return exec.Command("open", u).Run()
}

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

// PlayFromHistory relance un morceau de la pile ; les disques posés au-dessus sont retirés.
// Si la lecture échoue, la pile est remise comme avant.
func (s *VinylService) PlayFromHistory(source, trackID string) error {
	if s.history == nil {
		return errors.New("historique indisponible")
	}
	undo := s.history.TakeFrom(source, trackID)
	var err error
	if r, ok := s.watcher.Player(source).(player.ContextReplayer); ok {
		// On revient en arrière dans la playlist en cours : la suite continue après ce titre.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		err = r.ReplayInContext(ctx, trackID)
		cancel()
		s.watcher.Wake()
	} else {
		id := trackID
		if source == "applemusic" {
			id = "track:" + trackID // Musique attend un identifiant typé
		}
		err = s.PlayItem(source, id)
	}
	if err != nil {
		undo()
	}
	if s.onHistory != nil {
		s.onHistory(s.history.List())
	}
	return err
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
func (s *VinylService) PlayFrom(source, collectionID, trackID string, shuffle bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer s.watcher.Wake()
	switch p := s.watcher.Player(source).(type) {
	case *player.AppleMusic:
		if pid, ok := strings.CutPrefix(collectionID, "playlist:"); ok {
			return p.PlayInPlaylist(ctx, pid, trackID, shuffle)
		}
		col, err := s.Tracks(source, collectionID)
		if err != nil {
			return err
		}
		ids := make([]string, len(col.Tracks))
		for i, t := range col.Tracks {
			ids[i] = t.ID
		}
		return p.PlayList(ctx, ids, trackID, shuffle)
	case *player.Spotify:
		return p.PlayInContext(ctx, collectionID, trackID, shuffle)
	}
	return errors.New("source inconnue")
}

type QueueState struct {
	Shuffle  bool           `json:"shuffle"`
	Repeat   string         `json:"repeat"`
	Upcoming []player.Track `json:"upcoming"`
	Note     string         `json:"note"`
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
		web, ok := s.catalogs["spotify"].(*catalog.SpotifyWeb)
		if !ok || s.spotifyAuth == nil || !s.spotifyAuth.LoggedIn() {
			q.Note = "Connecte Spotify pour voir la suite"
			break
		}
		if ts, err := web.Queue(ctx); err == nil {
			q.Upcoming = ts
		} else {
			q.Note = "File Spotify indisponible"
		}
	}
	return q, nil
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
