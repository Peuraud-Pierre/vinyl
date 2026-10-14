// Package player pilote les lecteurs de musique macOS (Musique, Spotify) via AppleScript.
package player

import (
	"context"
	"errors"
	"time"
)

type State string

const (
	StatePlaying State = "playing"
	StatePaused  State = "paused"
	StateStopped State = "stopped"
)

var (
	ErrNotRunning    = errors.New("lecteur non lancé")
	ErrNotAuthorized = errors.New("automatisation non autorisée")
)

type NowPlaying struct {
	Source     string `json:"source"`
	TrackID    string `json:"trackId"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	Album      string `json:"album"`
	DurationMs int64  `json:"durationMs"`
	PositionMs int64  `json:"positionMs"`
	State      State  `json:"state"`
	ArtworkURL string `json:"artworkUrl"`
}

type Player interface {
	Name() string
	NowPlaying(ctx context.Context) (NowPlaying, error)
	PlayPause(ctx context.Context) error
	Next(ctx context.Context) error
	Previous(ctx context.Context) error
	Seek(ctx context.Context, pos time.Duration) error
	PlayItem(ctx context.Context, id string) error
}

func parseState(s string) State {
	switch s {
	case "playing":
		return StatePlaying
	case "paused":
		return StatePaused
	default:
		return StateStopped
	}
}

// Track est un titre d'album, de playlist ou de file d'attente.
type Track struct {
	ID         string `json:"id"` // persistent ID (Musique) ou URI spotify:track:…
	Number     int    `json:"number"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	DurationMs int64  `json:"durationMs"`
	Loved      bool   `json:"loved"`
}

// ParseSeconds est exporté pour les autres paquets (catalog) : secondes AppleScript → ms.
func ParseSeconds(s string) int64 { return parseSeconds(s) }
