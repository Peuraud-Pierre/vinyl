// Package winmedia pilote les lecteurs sous Windows par les contrôles multimédia du système (GSMTC).
// La logique (player.go) est testée partout ; seule la couche WinRT (sessions_windows.go) est propre
// à Windows.
package winmedia

import (
	"time"

	"vinyl/internal/player"
)

// Session : une app qui joue, vue au travers des contrôles multimédia de Windows.
type Session interface {
	AppID() string // SourceAppUserModelId, ex. « Spotify.exe »
	Info() (Info, error)
	Thumbnail() ([]byte, error) // miniature de la pochette ; nil si aucune
	TogglePlayPause() error
	Next() error
	Previous() error
	Seek(pos time.Duration) error
	SetShuffle(on bool) error
	SetRepeat(mode player.RepeatMode) error
}

// Info : état d'une session.
type Info struct {
	Title, Artist, Album string
	Position, Duration   time.Duration
	Status               player.State
	Shuffle              bool
	Repeat               player.RepeatMode
	UpdatedAt            time.Time // instant du dernier relevé de position par Windows
}

// Sessions liste les sessions ouvertes.
type Sessions interface {
	List() ([]Session, error)
}
