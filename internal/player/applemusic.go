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
	tell application id "com.apple.Music" to play (first playlist whose persistent ID is (item 1 of argv))
end run`

const musicPlayTrackScript = `on run argv
	tell application id "com.apple.Music" to play (first track of library playlist 1 whose persistent ID is (item 1 of argv))
end run`

// TempPlaylistName est la playlist que Vinyl remplit pour lire un album ou une liste de titres
// dans l'ordre ; ce n'est pas une playlist de l'utilisateur (masquée de l'étagère).
const TempPlaylistName = "Vinyl — Album"

const musicPlayAlbumScript = `on run argv
	set albumName to item 1 of argv
	set albumArtist to item 2 of argv
	tell application id "com.apple.Music"
		if not (exists user playlist "` + TempPlaylistName + `") then make new user playlist with properties {name:"` + TempPlaylistName + `"}
		set p to user playlist "` + TempPlaylistName + `"
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

// argv : titre de départ (vide = début), aléatoire (1/0), puis les persistent ID dans l'ordre.
const musicPlayListScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	set startID to item 1 of argv
	tell application id "com.apple.Music"
		if not (exists user playlist "` + TempPlaylistName + `") then make new user playlist with properties {name:"` + TempPlaylistName + `"}
		set p to user playlist "` + TempPlaylistName + `"
		delete every track of p
		repeat with i from 3 to count of argv
			try
				duplicate (first track of library playlist 1 whose persistent ID is (item i of argv)) to p
			end try
		end repeat
		set shuffle enabled to ((item 2 of argv) is "1")
		if startID is "" then
			play p
		else
			play (first track of p whose persistent ID is startID)
		end if
	end tell
end run`

// argv : playlist, titre de départ (vide = début), aléatoire (1/0).
const musicPlayInPlaylistScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	tell application id "com.apple.Music"
		set p to (first playlist whose persistent ID is (item 1 of argv))
		set shuffle enabled to ((item 3 of argv) is "1")
		if (item 2 of argv) is "" then
			play p
		else
			play (first track of p whose persistent ID is (item 2 of argv))
		end if
	end tell
end run`

func flag(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// PlayList remplit la playlist temporaire « Vinyl — Album » avec ids (dans l'ordre) puis joue à partir de startID.
func (m *AppleMusic) PlayList(ctx context.Context, ids []string, startID string, shuffle bool) error {
	out, err := m.run.Run(ctx, musicPlayListScript, append([]string{startID, flag(shuffle)}, ids...)...)
	if err == nil && out == "NOTRUNNING" {
		return ErrNotRunning
	}
	return err
}

// PlayInPlaylist joue une playlist de la bibliothèque à partir d'un titre ; la suite continue dans la playlist.
func (m *AppleMusic) PlayInPlaylist(ctx context.Context, playlistID, startID string, shuffle bool) error {
	out, err := m.run.Run(ctx, musicPlayInPlaylistScript, playlistID, startID, flag(shuffle))
	if err == nil && out == "NOTRUNNING" {
		return ErrNotRunning
	}
	return err
}

const musicModesScript = musicGuard + `tell application id "com.apple.Music"
	return ((shuffle enabled) as string) & (character id 31) & ((song repeat) as string)
end tell`

const musicSetShuffleScript = `on run argv
	tell application id "com.apple.Music" to set shuffle enabled to ((item 1 of argv) is "1")
end run`

const musicSetRepeatScript = `on run argv
	tell application id "com.apple.Music"
		if (item 1 of argv) is "one" then
			set song repeat to one
		else if (item 1 of argv) is "all" then
			set song repeat to all
		else
			set song repeat to off
		end if
	end tell
end run`

// argv : limite, reprise au début (1/0). Sortie : "index␟total␞" puis un enregistrement par titre.
const musicUpcomingScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	set us to character id 31
	set rs to character id 30
	set lim to (item 1 of argv) as integer
	tell application id "com.apple.Music"
		try
			set cp to current playlist
			set idx to index of current track
		on error
			return "NONE"
		end try
		set n to count of tracks of cp
		set out to (idx as string) & us & (n as string) & rs
		set k to 0
		set i to idx + 1
		repeat while k < lim and k < n - 1
			if i > n then
				if (item 2 of argv) is not "1" then exit repeat
				set i to 1
			end if
			set t to track i of cp
			set out to out & (persistent ID of t) & us & (name of t) & us & (artist of t) & us & ((duration of t) as string) & us & (album of t) & rs
			set k to k + 1
			set i to i + 1
		end repeat
		return out
	end tell
end run`

func (m *AppleMusic) PlaybackModes(ctx context.Context) (bool, RepeatMode, error) {
	out, err := m.run.Run(ctx, musicModesScript)
	if err != nil {
		return false, RepeatOff, err
	}
	if out == "NOTRUNNING" {
		return false, RepeatOff, ErrNotRunning
	}
	sh, rep, _ := strings.Cut(out, Sep)
	return sh == "true", parseRepeat(rep), nil
}

func (m *AppleMusic) SetShuffle(ctx context.Context, on bool) error {
	_, err := m.run.Run(ctx, musicSetShuffleScript, flag(on))
	return err
}

func (m *AppleMusic) SetRepeat(ctx context.Context, mode RepeatMode) error {
	if mode != RepeatOff && mode != RepeatAll && mode != RepeatOne {
		return fmt.Errorf("applemusic: mode de répétition inconnu %q", mode)
	}
	_, err := m.run.Run(ctx, musicSetRepeatScript, string(mode))
	return err
}

// Upcoming renvoie les titres qui suivent dans la playlist en cours (nil si aucune : radio, flux).
func (m *AppleMusic) Upcoming(ctx context.Context, limit int, wrap bool) ([]Track, error) {
	out, err := m.run.Run(ctx, musicUpcomingScript, strconv.Itoa(limit), flag(wrap))
	if err != nil {
		return nil, err
	}
	return parseUpcoming(out)
}

// parseUpcoming lit la sortie des scripts de file : en-tête (index, nombre), puis un titre par ligne.
func parseUpcoming(out string) ([]Track, error) {
	switch out {
	case "NOTRUNNING":
		return nil, ErrNotRunning
	case "NONE", "":
		return nil, nil
	}
	recs := strings.Split(out, "\x1e")
	tracks := []Track{}
	for _, r := range recs[1:] {
		f := strings.Split(r, Sep)
		if len(f) == 5 {
			tracks = append(tracks, Track{ID: f[0], Title: f[1], Artist: f[2], DurationMs: parseSeconds(f[3]), Album: f[4], Number: len(tracks) + 1})
		}
	}
	return tracks, nil
}

// argv : playlist, morceau, limite, boucle (1/0). Titres d'une playlist à partir d'un morceau (inclus) :
// la suite qui reprendra après une réécoute.
const musicUpcomingFromScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	set us to character id 31
	set rs to character id 30
	set lim to (item 3 of argv) as integer
	tell application id "com.apple.Music"
		try
			set cp to (first playlist whose persistent ID is (item 1 of argv))
			set idx to index of (first track of cp whose persistent ID is (item 2 of argv))
		on error
			return "NONE"
		end try
		set n to count of tracks of cp
		set out to (idx as string) & us & (n as string) & rs
		set k to 0
		set i to idx
		repeat while k < lim and k < n
			if i > n then
				if (item 4 of argv) is not "1" then exit repeat
				set i to 1
			end if
			set t to track i of cp
			set out to out & (persistent ID of t) & us & (name of t) & us & (artist of t) & us & ((duration of t) as string) & us & (album of t) & rs
			set k to k + 1
			set i to i + 1
		end repeat
		return out
	end tell
end run`

// UpcomingFrom renvoie les titres d'une playlist à partir d'un morceau (inclus).
func (m *AppleMusic) UpcomingFrom(ctx context.Context, playlistID, trackID string, limit int, wrap bool) ([]Track, error) {
	out, err := m.run.Run(ctx, musicUpcomingFromScript, playlistID, trackID, strconv.Itoa(limit), flag(wrap))
	if err != nil {
		return nil, err
	}
	return parseUpcoming(out)
}

// ReplayPlaylistName est la playlist interne où Vinyl joue seul un morceau repris de la pile (rien ne
// s'enchaîne derrière lui) ; distincte de TempPlaylistName pour ne pas écraser un album en cours.
const ReplayPlaylistName = "Vinyl — Réécoute"

// Où en est la lecture : playlist en cours, morceau, position. Lu avant une réécoute pour reprendre
// ensuite là où on en était.
const musicSaveContextScript = `if application id "com.apple.Music" is not running then return "NOTRUNNING"
tell application id "com.apple.Music"
	set us to character id 31
	set pl to ""
	try
		set pl to persistent ID of current playlist
	end try
	return pl & us & (persistent ID of current track) & us & ((player position) as string)
end tell`

// argv : persistent ID. Joue le morceau seul, dans sa playlist interne (sans toucher à l'aléatoire).
const musicPlayAloneScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	tell application id "com.apple.Music"
		if not (exists user playlist "` + ReplayPlaylistName + `") then make new user playlist with properties {name:"` + ReplayPlaylistName + `"}
		set p to user playlist "` + ReplayPlaylistName + `"
		delete every track of p
		duplicate (first track of library playlist 1 whose persistent ID is (item 1 of argv)) to p
		play (first track of p)
	end tell
	return "OK"
end run`

// argv : playlist, morceau, position (s). Reprend la playlist là où elle en était ; si elle n'existe
// plus, le morceau est lancé depuis la bibliothèque.
const musicResumeScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	tell application id "com.apple.Music"
		try
			set p to (first playlist whose persistent ID is (item 1 of argv))
			play (first track of p whose persistent ID is (item 2 of argv))
		on error
			play (first track of library playlist 1 whose persistent ID is (item 2 of argv))
		end try
		set player position to ((item 3 of argv) as integer)
	end tell
	return "OK"
end run`

func (m *AppleMusic) simpleRun(ctx context.Context, script string, args ...string) (string, error) {
	out, err := m.run.Run(ctx, script, args...)
	if err == nil && out == "NOTRUNNING" {
		return "", ErrNotRunning
	}
	return out, err
}

// SaveContext lit où en est la lecture (playlist, morceau, position).
func (m *AppleMusic) SaveContext(ctx context.Context) (PlaybackContext, error) {
	out, err := m.simpleRun(ctx, musicSaveContextScript)
	if err != nil {
		return PlaybackContext{}, err
	}
	f := strings.Split(out, Sep)
	if len(f) != 3 {
		return PlaybackContext{}, fmt.Errorf("applemusic: sortie inattendue (%d champs)", len(f))
	}
	return PlaybackContext{PlaylistID: f[0], TrackID: f[1], PositionSec: int(parseSeconds(f[2]) / 1000)}, nil
}

// PlayAlone joue un morceau seul : à sa fin, rien ne s'enchaîne (la reprise est faite par Vinyl).
func (m *AppleMusic) PlayAlone(ctx context.Context, trackID string) error {
	_, err := m.simpleRun(ctx, musicPlayAloneScript, trackID)
	return err
}

// Resume reprend la playlist là où elle en était.
func (m *AppleMusic) Resume(ctx context.Context, c PlaybackContext) error {
	_, err := m.simpleRun(ctx, musicResumeScript, c.PlaylistID, c.TrackID, strconv.Itoa(c.PositionSec))
	return err
}
