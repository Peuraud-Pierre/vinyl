package winmedia

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"vinyl/internal/player"
)

// Source : une app reconnue par l'identifiant de sa session, sous le nom de source de Vinyl.
type Source struct {
	Name  string
	Match func(appID string) bool
}

var (
	// SpotifySource : Spotify classique (Spotify.exe) ou du Microsoft Store (SpotifyAB.SpotifyMusic_…).
	SpotifySource = Source{Name: "spotify", Match: func(id string) bool {
		l := strings.ToLower(id)
		return l == "spotify.exe" || strings.HasPrefix(l, "spotifyab.spotifymusic")
	}}
	// AppleMusicSource : Apple Music pour Windows (AppleInc.AppleMusicWin_…).
	AppleMusicSource = Source{Name: "applemusic", Match: func(id string) bool {
		return strings.HasPrefix(strings.ToLower(id), "appleinc.applemusic")
	}}
)

// TrackID : Windows ne donne pas d'identifiant de morceau ; empreinte stable titre + artiste + album.
func TrackID(title, artist, album string) string {
	sum := sha1.Sum([]byte(title + "\x1f" + artist + "\x1f" + album))
	return "w" + hex.EncodeToString(sum[:8])
}

// Player : un lecteur de Vinyl (Spotify ou Apple Music) sous Windows.
type Player struct {
	src      Source
	sessions Sessions
	now      func() time.Time
}

func NewPlayer(src Source, s Sessions, now func() time.Time) *Player {
	return &Player{src: src, sessions: s, now: now}
}

func (p *Player) Name() string { return p.src.Name }

// session : la session de l'app ; aucune → app non lancée.
func (p *Player) session() (Session, error) {
	list, err := p.sessions.List()
	if err != nil {
		return nil, err
	}
	for _, s := range list {
		if p.src.Match(s.AppID()) {
			return s, nil
		}
	}
	return nil, player.ErrNotRunning
}

func (p *Player) NowPlaying(context.Context) (player.NowPlaying, error) {
	s, err := p.session()
	if err != nil {
		return player.NowPlaying{}, err
	}
	in, err := s.Info()
	if err != nil {
		return player.NowPlaying{}, err
	}
	if in.Title == "" { // publicité, changement de morceau
		return player.NowPlaying{Source: p.src.Name, State: player.StateStopped}, nil
	}
	pos := in.Position
	if in.Status == player.StatePlaying && !in.UpdatedAt.IsZero() {
		pos += p.now().Sub(in.UpdatedAt) // Windows ne relève la position que de temps en temps
	}
	if in.Duration > 0 && pos > in.Duration {
		pos = in.Duration
	}
	return player.NowPlaying{
		Source: p.src.Name, State: in.Status, TrackID: TrackID(in.Title, in.Artist, in.Album),
		Title: in.Title, Artist: in.Artist, Album: in.Album,
		DurationMs: in.Duration.Milliseconds(), PositionMs: pos.Milliseconds(),
	}, nil
}

func (p *Player) do(f func(Session) error) error {
	s, err := p.session()
	if err != nil {
		return err
	}
	return f(s)
}

func (p *Player) PlayPause(context.Context) error { return p.do(Session.TogglePlayPause) }
func (p *Player) Next(context.Context) error      { return p.do(Session.Next) }
func (p *Player) Previous(context.Context) error  { return p.do(Session.Previous) }
func (p *Player) Seek(_ context.Context, pos time.Duration) error {
	return p.do(func(s Session) error { return s.Seek(pos) })
}

// PlayItem : les contrôles de Windows ne savent pas lancer un album ; Spotify passe par l'API web (service).
func (p *Player) PlayItem(context.Context, string) error {
	return errors.New("non disponible sous Windows")
}

func (p *Player) PlaybackModes(context.Context) (bool, player.RepeatMode, error) {
	s, err := p.session()
	if err != nil {
		return false, player.RepeatOff, err
	}
	in, err := s.Info()
	return in.Shuffle, in.Repeat, err
}

func (p *Player) SetShuffle(_ context.Context, on bool) error {
	return p.do(func(s Session) error { return s.SetShuffle(on) })
}

func (p *Player) SetRepeat(_ context.Context, m player.RepeatMode) error {
	return p.do(func(s Session) error { return s.SetRepeat(m) })
}
