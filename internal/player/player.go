// Package player pilote les lecteurs de musique macOS (Musique, Spotify) via AppleScript.
package player

import (
	"context"
	"errors"
	"strings"
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
	Album      string `json:"album"`
	DurationMs int64  `json:"durationMs"`
	Loved      bool   `json:"loved"`
}

// ParseSeconds est exporté pour les autres paquets (catalog) : secondes AppleScript → ms.
func ParseSeconds(s string) int64 { return parseSeconds(s) }

type RepeatMode string

const (
	RepeatOff RepeatMode = "off"
	RepeatAll RepeatMode = "all"
	RepeatOne RepeatMode = "one"
)

// PlaybackContext dit où en est la lecture : playlist en cours, morceau, position (s).
type PlaybackContext struct {
	PlaylistID  string
	TrackID     string
	PositionSec int
	After       bool // reprendre au morceau qui suit TrackID (comme après un clic dans l'Historique)
}

// ContextSaver est implémentée par les lecteurs qui savent dire où en est la lecture (morceau,
// position), pour y revenir après une réécoute depuis la pile.
type ContextSaver interface {
	SaveContext(ctx context.Context) (PlaybackContext, error)
}

// ReplayResumer est implémentée par les lecteurs qui savent réécouter un morceau comme Apple Music :
// retenir où en est la playlist, jouer le morceau seul, puis reprendre la playlist là où elle en était.
type ReplayResumer interface {
	SaveContext(ctx context.Context) (PlaybackContext, error)
	PlayAlone(ctx context.Context, trackID string) error
	Resume(ctx context.Context, c PlaybackContext) error
}

// ContextPlayer est implémentée par les lecteurs qui savent lancer un morceau sans vider leur file : dans
// une playlist (repli : la bibliothèque), comme un clic dans Musique.
type ContextPlayer interface {
	PlayInContext(ctx context.Context, playlistID, trackID string) error
}

// Modes est implémentée par les lecteurs qui savent lire et régler l'aléatoire et la répétition.
type Modes interface {
	PlaybackModes(ctx context.Context) (shuffle bool, repeat RepeatMode, err error)
	SetShuffle(ctx context.Context, on bool) error
	SetRepeat(ctx context.Context, mode RepeatMode) error
}

func parseRepeat(s string) RepeatMode {
	switch RepeatMode(strings.TrimSpace(s)) {
	case RepeatAll:
		return RepeatAll
	case RepeatOne:
		return RepeatOne
	}
	return RepeatOff
}

// BackReplayer est implémentée par les lecteurs qui savent naviguer dans leur propre file
// (précédent / suivant) : réécouter un morceau déjà passé sans remplacer la file, comme Apple Music.
type BackReplayer interface {
	ReplayBack(ctx context.Context, trackID string, steps, posSec int) (taken int, err error)
	JumpForward(ctx context.Context, steps int, trackID string, posSec int) error
}

// IsRunning dit si une application est lancée (branché sur macOS par main) ; une app non lancée n'est
// pas interrogée, ce qui évite de lancer un osascript à chaque sondage. Par défaut : supposée lancée.
var IsRunning = func(bundleID string) bool { return true }
