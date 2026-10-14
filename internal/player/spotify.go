package player

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const spotifyGuard = `if application id "com.spotify.client" is not running then return "NOTRUNNING"
`

// Note : "st" est réservé en AppleScript (ordinal 1st) — d'où "ps".
const spotifyNowPlayingScript = spotifyGuard + `tell application id "com.spotify.client"
	set ps to player state as string
	if ps is "stopped" then return "stopped"
	set t to current track
	set us to character id 31
	set art to ""
	try
		set art to artwork url of t
	end try
	return ps & us & (id of t) & us & (name of t) & us & (artist of t) & us & (album of t) & us & ((duration of t) as string) & us & ((player position) as string) & us & art
end tell`

type Spotify struct{ run Runner }

func NewSpotify(r Runner) *Spotify { return &Spotify{run: r} }

func (s *Spotify) Name() string { return "spotify" }

func (s *Spotify) NowPlaying(ctx context.Context) (NowPlaying, error) {
	out, err := s.run.Run(ctx, spotifyNowPlayingScript)
	if err != nil {
		return NowPlaying{}, err
	}
	switch out {
	case "NOTRUNNING":
		return NowPlaying{}, ErrNotRunning
	case "stopped":
		return NowPlaying{Source: s.Name(), State: StateStopped}, nil
	}
	f := strings.Split(out, Sep)
	if len(f) != 8 {
		return NowPlaying{}, fmt.Errorf("spotify: sortie inattendue (%d champs)", len(f))
	}
	art := f[7]
	if !strings.HasPrefix(art, "http") {
		art = ""
	}
	return NowPlaying{
		Source: s.Name(), State: parseState(f[0]), TrackID: f[1], Title: f[2], Artist: f[3], Album: f[4],
		DurationMs: parseMillis(f[5]), PositionMs: parseSeconds(f[6]), ArtworkURL: art,
	}, nil
}

func (s *Spotify) simple(ctx context.Context, cmd string) error {
	_, err := s.run.Run(ctx, spotifyGuard+`tell application id "com.spotify.client" to `+cmd)
	return err
}

func (s *Spotify) PlayPause(ctx context.Context) error { return s.simple(ctx, "playpause") }
func (s *Spotify) Next(ctx context.Context) error      { return s.simple(ctx, "next track") }
func (s *Spotify) Previous(ctx context.Context) error  { return s.simple(ctx, "previous track") }

func (s *Spotify) Seek(ctx context.Context, pos time.Duration) error {
	if pos < 0 {
		pos = 0
	}
	_, err := s.run.Run(ctx, `on run argv
	tell application id "com.spotify.client" to set player position to ((item 1 of argv) as integer)
end run`, strconv.FormatInt(int64(pos/time.Second), 10))
	return err
}

func (s *Spotify) PlayItem(ctx context.Context, uri string) error {
	if !strings.HasPrefix(uri, "spotify:") {
		return fmt.Errorf("spotify: URI invalide %q", uri)
	}
	_, err := s.run.Run(ctx, `on run argv
	tell application id "com.spotify.client" to play track (item 1 of argv)
end run`, uri)
	return err
}
