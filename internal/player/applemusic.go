package player

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const musicGuard = `if application id "com.apple.Music" is not running then return "NOTRUNNING"
`

const musicNowPlayingScript = musicGuard + `tell application id "com.apple.Music"
	set ps to player state as string
	if ps is "stopped" then return "stopped"
	try
		set t to current track
	on error
		return "stopped"
	end try
	set us to character id 31
	set d to "0"
	try
		set d to (duration of t) as string
	end try
	return ps & us & (persistent ID of t) & us & (name of t) & us & (artist of t) & us & (album of t) & us & d & us & ((player position) as string) & us & ((count of artworks of t) as string)
end tell`

// argv : chemin du fichier, persistent ID attendu. Renvoie "CHANGED" si le morceau a changé entre-temps.
const musicExportArtworkScript = `on run argv
	tell application id "com.apple.Music"
		if (persistent ID of current track) is not (item 2 of argv) then return "CHANGED"
		set img to raw data of artwork 1 of current track
	end tell
	set f to open for access (POSIX file (item 1 of argv)) with write permission
	try
		set eof f to 0
		write img to f
	on error
		close access f
		return "FAILED"
	end try
	close access f
	return "OK"
end run`

const musicPlayPlaylistScript = `on run argv
	tell application id "com.apple.Music" to play (first user playlist whose persistent ID is (item 1 of argv))
end run`

const musicPlayTrackScript = `on run argv
	tell application id "com.apple.Music" to play (first track of library playlist 1 whose persistent ID is (item 1 of argv))
end run`

const musicPlayAlbumScript = `on run argv
	set albumName to item 1 of argv
	set albumArtist to item 2 of argv
	tell application id "com.apple.Music"
		if not (exists user playlist "Vinyl — Album") then make new user playlist with properties {name:"Vinyl — Album"}
		set p to user playlist "Vinyl — Album"
		delete every track of p
		set ts to (every track of library playlist 1 whose album is albumName and album artist is albumArtist)
		if (count of ts) is 0 then set ts to (every track of library playlist 1 whose album is albumName)
		repeat with t in ts
			duplicate t to p
		end repeat
		play p
	end tell
end run`

const musicPlayLibraryTrackScript = `on run argv
	tell application id "com.apple.Music"
		set ts to (every track of library playlist 1 whose name is (item 1 of argv) and artist is (item 2 of argv))
		if (count of ts) is 0 then return "NOTFOUND"
		play item 1 of ts
		return "OK"
	end tell
end run`

type AppleMusic struct {
	run        Runner
	artworkDir string
}

func NewAppleMusic(r Runner, artworkDir string) *AppleMusic {
	return &AppleMusic{run: r, artworkDir: artworkDir}
}

func (m *AppleMusic) Name() string { return "applemusic" }

func ArtworkFileName(trackID string) string {
	sum := sha1.Sum([]byte(trackID))
	return hex.EncodeToString(sum[:]) + ".img"
}

func (m *AppleMusic) NowPlaying(ctx context.Context) (NowPlaying, error) {
	out, err := m.run.Run(ctx, musicNowPlayingScript)
	if err != nil {
		return NowPlaying{}, err
	}
	switch out {
	case "NOTRUNNING":
		return NowPlaying{}, ErrNotRunning
	case "stopped":
		return NowPlaying{Source: m.Name(), State: StateStopped}, nil
	}
	f := strings.Split(out, Sep)
	if len(f) != 8 {
		return NowPlaying{}, fmt.Errorf("applemusic: sortie inattendue (%d champs)", len(f))
	}
	np := NowPlaying{
		Source: m.Name(), State: parseState(f[0]), TrackID: f[1], Title: f[2], Artist: f[3], Album: f[4],
		DurationMs: parseSeconds(f[5]), PositionMs: parseSeconds(f[6]),
	}
	if n, _ := strconv.Atoi(strings.TrimSpace(f[7])); n > 0 && np.TrackID != "" {
		if err := m.ensureArtwork(ctx, np.TrackID); err == nil {
			np.ArtworkURL = "/artwork/" + ArtworkFileName(np.TrackID)
		}
	}
	return np, nil
}

func (m *AppleMusic) ensureArtwork(ctx context.Context, trackID string) error {
	path := filepath.Join(m.artworkDir, ArtworkFileName(trackID))
	if fi, err := os.Stat(path); err == nil && fi.Size() > 0 {
		return nil
	}
	if err := os.MkdirAll(m.artworkDir, 0o755); err != nil {
		return err
	}
	// Export dans un fichier temporaire, renommé seulement s'il est complet et du bon morceau.
	tmp := path + ".tmp"
	defer os.Remove(tmp)
	out, err := m.run.Run(ctx, musicExportArtworkScript, tmp, trackID)
	if err != nil {
		return err
	}
	if fi, err := os.Stat(tmp); out != "OK" || err != nil || fi.Size() == 0 {
		return fmt.Errorf("applemusic: export de pochette %s (%s)", trackID, out)
	}
	return os.Rename(tmp, path)
}

func (m *AppleMusic) simple(ctx context.Context, cmd string) error {
	_, err := m.run.Run(ctx, musicGuard+`tell application id "com.apple.Music" to `+cmd)
	return err
}

func (m *AppleMusic) PlayPause(ctx context.Context) error { return m.simple(ctx, "playpause") }
func (m *AppleMusic) Next(ctx context.Context) error      { return m.simple(ctx, "next track") }
func (m *AppleMusic) Previous(ctx context.Context) error  { return m.simple(ctx, "previous track") }

func (m *AppleMusic) Seek(ctx context.Context, pos time.Duration) error {
	if pos < 0 {
		pos = 0
	}
	// Entier en argv : la conversion texte → réel d'AppleScript dépend de la locale.
	_, err := m.run.Run(ctx, `on run argv
	tell application id "com.apple.Music" to set player position to ((item 1 of argv) as integer)
end run`, strconv.FormatInt(int64(pos/time.Second), 10))
	return err
}

func (m *AppleMusic) PlayItem(ctx context.Context, id string) error {
	kind, rest, ok := strings.Cut(id, ":")
	if !ok {
		return fmt.Errorf("applemusic: ID invalide %q", id)
	}
	var err error
	switch kind {
	case "playlist":
		_, err = m.run.Run(ctx, musicPlayPlaylistScript, rest)
	case "track":
		_, err = m.run.Run(ctx, musicPlayTrackScript, rest)
	case "album":
		album, artist, _ := strings.Cut(rest, Sep)
		_, err = m.run.Run(ctx, musicPlayAlbumScript, album, artist)
	default:
		return fmt.Errorf("applemusic: type d'ID inconnu %q", kind)
	}
	return err
}

// PlayLibraryTrack lance un titre de la bibliothèque par nom et artiste. Renvoie false s'il est absent.
func (m *AppleMusic) PlayLibraryTrack(ctx context.Context, title, artist string) (bool, error) {
	out, err := m.run.Run(ctx, musicPlayLibraryTrackScript, title, artist)
	if err != nil {
		return false, err
	}
	return out == "OK", nil
}
