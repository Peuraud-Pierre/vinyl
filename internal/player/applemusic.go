package player

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
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
	// HD télécharge dans dst la pochette HD (iTunes) d'un album ; nil : pas de pochette HD.
	HD     func(ctx context.Context, artist, album, dst string) bool
	hdSeen sync.Map // morceau → struct{} : pochette HD déjà cherchée (une fois par morceau)
}

func NewAppleMusic(r Runner, artworkDir string) *AppleMusic {
	return &AppleMusic{run: r, artworkDir: artworkDir}
}

func (m *AppleMusic) Name() string { return "applemusic" }

func ArtworkFileName(trackID string) string {
	sum := sha1.Sum([]byte(trackID))
	return hex.EncodeToString(sum[:]) + ".img"
}

// HDFileName : fichier de la pochette HD (iTunes) d'un morceau.
func HDFileName(trackID string) string {
	sum := sha1.Sum([]byte(trackID))
	return hex.EncodeToString(sum[:]) + ".hd.jpg"
}

func (m *AppleMusic) NowPlaying(ctx context.Context) (NowPlaying, error) {
	if !IsRunning("com.apple.Music") {
		return NowPlaying{}, ErrNotRunning
	}
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
	if np.TrackID == "" {
		return np, nil
	}
	// Pochette HD (iTunes) si elle est là ; sinon celle de Musique tout de suite, et la HD cherchée en
	// arrière-plan : elle la remplacera au sondage suivant.
	if fi, err := os.Stat(filepath.Join(m.artworkDir, HDFileName(np.TrackID))); err == nil && fi.Size() > 0 {
		np.ArtworkURL = "/artwork/" + HDFileName(np.TrackID)
		return np, nil
	}
	if n, _ := strconv.Atoi(strings.TrimSpace(f[7])); n > 0 {
		if err := m.ensureArtwork(ctx, np.TrackID); err == nil {
			np.ArtworkURL = "/artwork/" + ArtworkFileName(np.TrackID)
		}
	}
	m.upgradeHD(np.TrackID, np.Artist, np.Album)
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

// argv : persistent ID. Joue le morceau une seule fois puis s'arrête (« with once ») : aucune playlist
// n'est créée, Vinyl enchaîne lui-même la suite. Absent de la bibliothèque (écouté en streaming) : pris
// dans la playlist en cours.
const musicPlayAloneScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	tell application id "com.apple.Music"
		try
			set t to (first track of library playlist 1 whose persistent ID is (item 1 of argv))
		on error
			set t to (first track of current playlist whose persistent ID is (item 1 of argv))
		end try
		play t with once
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

// PlayAlone joue un morceau une seule fois : à sa fin, Musique s'arrête et Vinyl enchaîne la suite.
func (m *AppleMusic) PlayAlone(ctx context.Context, trackID string) error {
	_, err := m.simpleRun(ctx, musicPlayAloneScript, trackID)
	return err
}

// argv : persistent ID. « Lire ensuite » comme à la main, la seule façon de jouer un morceau sans
// remplacer la file de Musique (aucune commande AppleScript ne le fait) : le morceau est sélectionné
// dans la liste de la playlist en cours (fenêtre de Musique), Song › Play Next (Morceau › Lire ensuite)
// est choisi au menu, puis « suivant » : le morceau en cours est sauté, le morceau choisi joue et la
// file continue ensuite. Musique passe au premier plan le temps du clic, puis l'app d'avant y revient.
// Demande l'autorisation Accessibilité.
const musicPlayNextScript = `on rowHas(r, nm)
	tell application "System Events"
		repeat with c in (UI elements of r)
			repeat with x in (static texts of c)
				try
					if ((value of x) as text) starts with nm then return true
				end try
			end repeat
		end repeat
	end tell
	return false
end rowHas

-- Liste des morceaux affichée (directement dans la fenêtre, ou dans un groupe selon la page).
on trackTable()
	tell application "System Events" to tell process "Music"
		repeat with g in (UI elements of splitter group 1 of window 1)
			try
				if role of g is "AXScrollArea" then
					if (count of tables of g) > 0 then return table 1 of g
				else if role of g is "AXGroup" then
					repeat with sa in (scroll areas of g)
						if (count of tables of sa) > 0 then return table 1 of sa
					end repeat
				end if
			end try
		end repeat
	end tell
	return missing value
end trackTable

on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	set target to item 1 of argv
	tell application "System Events" to set prev to name of first application process whose frontmost is true
	tell application id "com.apple.Music"
		try
			set t to (first track of current playlist whose persistent ID is target)
		on error
			return "NOTFOUND"
		end try
		set idx to index of t
		set nm to name of t
		set plName to name of current playlist
		try
			set minimized of front browser window to false
		end try
	end tell
	-- Ouvre la playlist en cours : clic sur son nom dans la barre latérale (« set view » ne change plus
	-- de page dans les versions récentes de Musique).
	tell application "System Events" to tell process "Music"
		repeat with r in (rows of outline 1 of scroll area 1 of splitter group 1 of window 1)
			try
				if (name of UI element 1 of r) is plName then
					set selected of r to true
					exit repeat
				end if
			end try
		end repeat
	end tell
	set tb to missing value
	repeat 50 times
		set tb to my trackTable()
		if tb is not missing value then
			tell application "System Events" to if (count of rows of tb) ≥ idx then exit repeat
		end if
		delay 0.1
	end repeat
	if tb is missing value then return "NOTFOUND"
	set res to "OK"
	tell application "System Events" to tell process "Music"
		set r to missing value
		if idx ≤ (count of rows of tb) then
			if my rowHas(row idx of tb, nm) then set r to row idx of tb
		end if
		if r is missing value then
			repeat with x in (rows of tb)
				if my rowHas(x, nm) then
					set r to x
					exit repeat
				end if
			end repeat
		end if
		if r is missing value then return "NOTFOUND"
		set selected of r to true
		set frontmost to true
		delay 0.3
		set m to menu 1 of menu bar item 5 of menu bar 1
		set mi to missing value
		repeat with label in {"Play Next", "Lire ensuite"}
			if exists menu item (label as text) of m then
				set mi to menu item (label as text) of m
				exit repeat
			end if
		end repeat
		if mi is missing value then set mi to menu item 5 of m
		if enabled of mi then
			click mi
		else
			set res to "DISABLED"
		end if
	end tell
	if res is "OK" then
		delay 0.3
		tell application id "com.apple.Music" to next track
	end if
	if prev is not "Music" then
		try
			tell application "System Events" to set frontmost of process prev to true
		end try
	end if
	return res
end run`

// ErrNotInPlaylist : le morceau n'est pas dans la playlist en cours (« Lire ensuite » impossible).
var ErrNotInPlaylist = errors.New("ce morceau n'est pas dans la playlist en cours")

// ErrNeedsAccessibility : Vinyl n'a pas l'autorisation Accessibilité (pilotage de la fenêtre de Musique).
var ErrNeedsAccessibility = errors.New("autorise Vinyl dans Réglages Système › Confidentialité et sécurité › Accessibilité")

// PlayNext joue le morceau tout de suite sans vider la file de Musique (Lire ensuite, puis suivant).
func (m *AppleMusic) PlayNext(ctx context.Context, trackID string) error {
	out, err := m.simpleRun(ctx, musicPlayNextScript, trackID)
	switch {
	case err != nil && (strings.Contains(err.Error(), "-25211") || strings.Contains(err.Error(), "-1719") || strings.Contains(err.Error(), "assistive")):
		return ErrNeedsAccessibility
	case err != nil:
		return err
	case out == "NOTFOUND":
		return ErrNotInPlaylist
	case out == "DISABLED":
		return errors.New("Musique n'a pas accepté « Lire ensuite »")
	}
	return nil
}

// upgradeHD cherche une fois, en arrière-plan, la pochette HD du morceau (sans retarder le sondage).
func (m *AppleMusic) upgradeHD(trackID, artist, album string) {
	if m.HD == nil || album == "" {
		return
	}
	if _, seen := m.hdSeen.LoadOrStore(trackID, struct{}{}); seen {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		m.HD(ctx, artist, album, filepath.Join(m.artworkDir, HDFileName(trackID)))
	}()
}

// argv : persistent ID. Paroles enregistrées dans Musique pour ce morceau (en cours, sinon dans la
// bibliothèque) ; "" s'il n'y en a pas.
const musicLyricsScript = `on run argv
	if application id "com.apple.Music" is not running then return ""
	tell application id "com.apple.Music"
		try
			set t to current track
			if (persistent ID of t) is (item 1 of argv) then return (lyrics of t) as text
		end try
		try
			return (lyrics of (first track of library playlist 1 whose persistent ID is (item 1 of argv))) as text
		end try
	end tell
	return ""
end run`

// Lyrics renvoie les paroles (texte simple) enregistrées dans Musique pour un morceau.
func (m *AppleMusic) Lyrics(ctx context.Context, trackID string) (string, error) {
	out, err := m.run.Run(ctx, musicLyricsScript, trackID)
	if err != nil || out == "missing value" {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// Favori d'un morceau (cœur de Musique : « favorited », « loved » sur les anciennes versions). Avec un
// 2e argument ("1" ou "0"), le règle d'abord. Le morceau en cours de préférence (il peut ne pas être
// dans la bibliothèque), sinon celui de la bibliothèque.
const musicFavoriteScript = `on run argv
	if application id "com.apple.Music" is not running then return "false"
	tell application id "com.apple.Music"
		set t to missing value
		try
			if (persistent ID of current track) is (item 1 of argv) then set t to current track
		end try
		if t is missing value then set t to (first track of library playlist 1 whose persistent ID is (item 1 of argv))
		if (count of argv) > 1 then
			set v to ((item 2 of argv) is "1")
			try
				set favorited of t to v
			on error
				set loved of t to v
			end try
		end if
		try
			return (favorited of t) as text
		on error
			return (loved of t) as text
		end try
	end tell
end run`

// Favorite dit si le morceau est en favori dans Musique.
func (m *AppleMusic) Favorite(ctx context.Context, trackID string) (bool, error) {
	out, err := m.run.Run(ctx, musicFavoriteScript, trackID)
	return strings.TrimSpace(out) == "true", err
}

// SetFavorite met le morceau en favori dans Musique, ou l'en retire.
func (m *AppleMusic) SetFavorite(ctx context.Context, trackID string, on bool) error {
	v := "0"
	if on {
		v = "1"
	}
	out, err := m.run.Run(ctx, musicFavoriteScript, trackID, v)
	if err == nil && (strings.TrimSpace(out) == "true") != on {
		return errors.New("Musique n'a pas pris le favori en compte")
	}
	return err
}
