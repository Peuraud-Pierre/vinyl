// Package settings gère les préférences de l'app dans un fichier JSON.
package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sync"
)

type Settings struct {
	Source           string    `json:"source"`
	Texture          string    `json:"texture"`
	DiscColors       [2]string `json:"discColors"`
	Accent           string    `json:"accent"`
	Crackle          bool      `json:"crackle"`
	WidgetX          int       `json:"widgetX"`
	WidgetY          int       `json:"widgetY"`
	WidgetPlaced     bool      `json:"widgetPlaced"`
	SpotifyClientID  string    `json:"spotifyClientId"`
	WidgetProjectDir string    `json:"widgetProjectDir,omitempty"` // dossier du dépôt (widget/), facultatif
}

func Defaults() Settings {
	return Settings{
		Source:     "auto",
		Texture:    "black",
		DiscColors: [2]string{"#ffb300", "#fff3d6"},
		Accent:     "#f5c542",
		Crackle:    true,
	}
}

var hexColor = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

// Normalize remplace toute valeur invalide par la valeur par défaut.
func Normalize(v Settings) Settings {
	d := Defaults()
	if !slices.Contains([]string{"auto", "applemusic", "spotify"}, v.Source) {
		v.Source = d.Source
	}
	if !slices.Contains([]string{"black", "picture", "yolk", "split", "solid"}, v.Texture) {
		v.Texture = d.Texture
	}
	for i := range v.DiscColors {
		if !hexColor.MatchString(v.DiscColors[i]) {
			v.DiscColors[i] = d.DiscColors[i]
		}
	}
	if !hexColor.MatchString(v.Accent) {
		v.Accent = d.Accent
	}
	return v
}

type Store struct {
	path string
	mu   sync.RWMutex
	cur  Settings
}

func NewStore(path string) *Store { return &Store{path: path, cur: Defaults()} }

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir() // ~/Library/Application Support
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Vinyl", "settings.json"), nil
}

// Load lit le fichier. Fichier absent ou illisible → valeurs par défaut, sans erreur.
// On décode par-dessus les défauts : un champ absent garde sa valeur par défaut.
func (s *Store) Load() error {
	v := Defaults()
	if data, err := os.ReadFile(s.path); err == nil {
		if json.Unmarshal(data, &v) != nil {
			v = Defaults()
		}
	}
	s.mu.Lock()
	s.cur = Normalize(v)
	s.mu.Unlock()
	return nil
}

func (s *Store) Get() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.cur
}

// Save remplace tous les réglages. Préférer Update depuis l'interface.
func (s *Store) Save(v Settings) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.writeLocked(Normalize(v))
}

// Patch décrit une mise à jour partielle : seuls les champs non nil sont modifiés.
type Patch struct {
	Source          *string    `json:"source,omitempty"`
	Texture         *string    `json:"texture,omitempty"`
	DiscColors      *[2]string `json:"discColors,omitempty"`
	Accent          *string    `json:"accent,omitempty"`
	Crackle         *bool      `json:"crackle,omitempty"`
	SpotifyClientID *string    `json:"spotifyClientId,omitempty"`
}

// Update applique un Patch aux réglages courants, sous verrou, et renvoie le résultat.
func (s *Store) Update(p Patch) (Settings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v := s.cur
	if p.Source != nil {
		v.Source = *p.Source
	}
	if p.Texture != nil {
		v.Texture = *p.Texture
	}
	if p.DiscColors != nil {
		v.DiscColors = *p.DiscColors
	}
	if p.Accent != nil {
		v.Accent = *p.Accent
	}
	if p.Crackle != nil {
		v.Crackle = *p.Crackle
	}
	if p.SpotifyClientID != nil {
		v.SpotifyClientID = *p.SpotifyClientID
	}
	v = Normalize(v)
	if err := s.writeLocked(v); err != nil {
		return s.cur, err
	}
	return v, nil
}

// writeLocked écrit v sur disque puis met à jour cur. Appelant : verrou en écriture tenu.
func (s *Store) writeLocked(v Settings) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return err
	}
	s.cur = v
	return nil
}
