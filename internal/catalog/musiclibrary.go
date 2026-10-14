package catalog

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"vinyl/internal/player"
)

const recordSep = "\x1e"

const musicGuard = `if application id "com.apple.Music" is not running then return "NOTRUNNING"
`

const playlistsScript = musicGuard + `tell application id "com.apple.Music"
	set us to character id 31
	set rs to character id 30
	set out to ""
	repeat with p in (every user playlist whose special kind is none)
		set out to out & (name of p) & us & (persistent ID of p) & us & ((count of tracks of p) as string) & rs
	end repeat
	-- Playlists ajoutées depuis le catalogue Apple Music (créées par Apple ou par d'autres).
	repeat with p in (every subscription playlist)
		set out to out & (name of p) & us & (persistent ID of p) & us & ((count of tracks of p) as string) & rs
	end repeat
	return out
end tell`

const albumsScript = musicGuard + `tell application id "com.apple.Music"
	set us to character id 31
	set rs to character id 30
	set als to album of every track of library playlist 1
	set ars to album artist of every track of library playlist 1
	set fallback to artist of every track of library playlist 1
	set out to ""
	repeat with i from 1 to count of als
		set ar to item i of ars
		if ar is "" then set ar to item i of fallback
		set out to out & (item i of als) & us & ar & rs
	end repeat
	return out
end tell`

const artistsScript = musicGuard + `tell application id "com.apple.Music"
	set rs to character id 30
	set AppleScript's text item delimiters to rs
	return (artist of every track of library playlist 1) as text
end tell`

// Exporte la pochette du premier titre d'un album ou d'une playlist. argv : kind, clé1, clé2, chemin
const itemArtworkScript = `on run argv
	set k to item 1 of argv
	tell application id "com.apple.Music"
		if k is "playlist" then
			set ts to tracks of (first playlist whose persistent ID is (item 2 of argv))
		else
			set ts to (every track of library playlist 1 whose album is (item 2 of argv))
		end if
		if (count of ts) is 0 then return "NONE"
		set t to item 1 of ts
		if (count of artworks of t) is 0 then return "NONE"
		set img to raw data of artwork 1 of t
	end tell
	set f to open for access (POSIX file (item 4 of argv)) with write permission
	try
		set eof f to 0
		write img to f
	end try
	close access f
	return "OK"
end run`

// maxArtworkExports limite les exports de pochettes simultanés : Musique traite les Apple Events
// en série, et trop d'appels en file retardent le sondage de la platine.
const maxArtworkExports = 2

type MusicLibrary struct {
	run        player.Runner
	itunes     *ITunes
	artworkDir string
	exports    chan struct{}
	noArtwork  sync.Map // id → struct{} : éléments sans pochette, pour ne pas relancer l'export
}

func NewMusicLibrary(r player.Runner, it *ITunes, artworkDir string) *MusicLibrary {
	return &MusicLibrary{run: r, itunes: it, artworkDir: artworkDir, exports: make(chan struct{}, maxArtworkExports)}
}

func (m *MusicLibrary) records(ctx context.Context, script string) ([][]string, error) {
	return m.recordsArgs(ctx, script)
}

func (m *MusicLibrary) recordsArgs(ctx context.Context, script string, args ...string) ([][]string, error) {
	out, err := m.run.Run(ctx, script, args...)
	if err != nil {
		return nil, err
	}
	if out == "NOTRUNNING" {
		return nil, player.ErrNotRunning
	}
	var recs [][]string
	for _, r := range strings.Split(out, recordSep) {
		if r != "" {
			recs = append(recs, strings.Split(r, player.Sep))
		}
	}
	return recs, nil
}

func sortByTitle(items []Item) {
	sort.Slice(items, func(i, j int) bool { return strings.ToLower(items[i].Title) < strings.ToLower(items[j].Title) })
}

func (m *MusicLibrary) Library(ctx context.Context, kind string, offset int) (Page, error) {
	var items []Item
	switch kind {
	case "playlists":
		recs, err := m.records(ctx, playlistsScript)
		if err != nil {
			return Page{}, err
		}
		for _, r := range recs {
			if len(r) == 3 {
				items = append(items, Item{ID: "playlist:" + r[1], Kind: "playlist", Title: r[0], Subtitle: r[2] + " titres"})
			}
		}
	case "albums":
		recs, err := m.records(ctx, albumsScript)
		if err != nil {
			return Page{}, err
		}
		seen := map[string]bool{}
		for _, r := range recs {
			if len(r) != 2 || r[0] == "" {
				continue
			}
			key := r[0] + player.Sep + r[1]
			if !seen[key] {
				seen[key] = true
				items = append(items, Item{ID: "album:" + key, Kind: "album", Title: r[0], Subtitle: r[1]})
			}
		}
		sortByTitle(items)
	case "artists":
		recs, err := m.records(ctx, artistsScript)
		if err != nil {
			return Page{}, err
		}
		seen := map[string]bool{}
		for _, r := range recs {
			name := strings.TrimSpace(r[0])
			if name != "" && !seen[name] {
				seen[name] = true
				items = append(items, Item{ID: name, Kind: "artist", Title: name})
			}
		}
		sortByTitle(items)
	default:
		return Page{}, ErrUnsupported
	}
	return paginate(items, offset), nil
}

func (m *MusicLibrary) Artist(ctx context.Context, name, _ string) (ArtistPage, error) {
	return m.itunes.Artist(ctx, name)
}

// Artwork exporte (une fois) la pochette d'un album ou d'une playlist de la bibliothèque.
func (m *MusicLibrary) Artwork(ctx context.Context, id string) (string, error) {
	if _, missing := m.noArtwork.Load(id); missing {
		return "", nil
	}
	file := player.ArtworkFileName("item:" + id)
	path := filepath.Join(m.artworkDir, file)
	if _, err := os.Stat(path); err == nil {
		return "/artwork/" + file, nil
	}
	select {
	case m.exports <- struct{}{}:
		defer func() { <-m.exports }()
	case <-ctx.Done():
		return "", ctx.Err()
	}
	kind, rest, _ := strings.Cut(id, ":")
	key1, key2, _ := strings.Cut(rest, player.Sep)
	if err := os.MkdirAll(m.artworkDir, 0o755); err != nil {
		return "", err
	}
	out, err := m.run.Run(ctx, itemArtworkScript, kind, key1, key2, path)
	if err != nil {
		return "", err
	}
	if out != "OK" {
		m.noArtwork.Store(id, struct{}{})
		return "", nil
	}
	return "/artwork/" + file, nil
}

// Champs : persistent ID, disque, piste, titre, artiste, durée, aimé (1/0), genre, année.
const trackFields = `(persistent ID of t) & us & ((disc number of t) as string) & us & ((track number of t) as string) & us & (name of t) & us & (artist of t) & us & ((duration of t) as string) & us & lv & us & (genre of t) & us & ((year of t) as string) & rs`

// « favorited » (macOS récents) ou « loved » (anciens) : un terme inconnu échoue à l'exécution, rattrapé par try.
const lovedSnippet = `
			set lv to "0"
			try
				if favorited of t then set lv to "1"
			on error
				try
					if loved of t then set lv to "1"
				end try
			end try`

// argv : album, artiste d'album (repli sur l'album seul si aucun titre ne correspond).
const albumTracksScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	set us to character id 31
	set rs to character id 30
	tell application id "com.apple.Music"
		set ts to (every track of library playlist 1 whose album is (item 1 of argv) and album artist is (item 2 of argv))
		if (count of ts) is 0 then set ts to (every track of library playlist 1 whose album is (item 1 of argv) and artist is (item 2 of argv))
		if (count of ts) is 0 then set ts to (every track of library playlist 1 whose album is (item 1 of argv))
		set out to ""
		repeat with t in ts` + lovedSnippet + `
			set out to out & ` + trackFields + `
		end repeat
		return out
	end tell
end run`

// argv : persistent ID de la playlist. Premier enregistrement : nom de la playlist.
const playlistTracksScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	set us to character id 31
	set rs to character id 30
	tell application id "com.apple.Music"
		set p to (first playlist whose persistent ID is (item 1 of argv))
		set out to (name of p) & rs
		repeat with t in (tracks of p)` + lovedSnippet + `
			set out to out & ` + trackFields + `
		end repeat
		return out
	end tell
end run`

func parseTrackRecord(r []string) (t player.Track, disc int, genre, year string, ok bool) {
	if len(r) != 9 {
		return t, 0, "", "", false
	}
	disc, _ = strconv.Atoi(strings.TrimSpace(r[1]))
	num, _ := strconv.Atoi(strings.TrimSpace(r[2]))
	year = strings.TrimSpace(r[8])
	if year == "0" {
		year = ""
	}
	t = player.Track{ID: r[0], Number: num, Title: r[3], Artist: r[4], DurationMs: player.ParseSeconds(r[5]), Loved: r[6] == "1"}
	return t, disc, strings.TrimSpace(r[7]), year, true
}

// Tracks renvoie un album (trié par disque puis piste) ou une playlist (dans son ordre) avec ses titres.
func (m *MusicLibrary) Tracks(ctx context.Context, id string) (Collection, error) {
	kind, rest, ok := strings.Cut(id, ":")
	if !ok || (kind != "album" && kind != "playlist") {
		return Collection{}, fmt.Errorf("musique : ID de collection invalide %q", id)
	}
	c := Collection{ID: id, Kind: kind, Source: "applemusic"}
	var recs [][]string
	var err error
	if kind == "album" {
		album, artist, _ := strings.Cut(rest, player.Sep)
		c.Title, c.Artist = album, artist
		recs, err = m.recordsArgs(ctx, albumTracksScript, album, artist)
	} else {
		recs, err = m.recordsArgs(ctx, playlistTracksScript, rest)
		if err == nil && len(recs) > 0 {
			c.Title = recs[0][0]
			recs = recs[1:]
		}
	}
	if err != nil {
		return Collection{}, err
	}
	type row struct {
		t    player.Track
		disc int
	}
	var rows []row
	for _, r := range recs {
		t, disc, genre, year, ok := parseTrackRecord(r)
		if !ok {
			continue
		}
		if c.Genre == "" {
			c.Genre = genre
		}
		if c.Year == "" {
			c.Year = year
		}
		if c.Artist == "" {
			c.Artist = t.Artist
		}
		rows = append(rows, row{t, disc})
	}
	if kind == "album" {
		// Disque puis piste ; une piste sans numéro (0) va en fin de son disque.
		order := func(n int) int {
			if n <= 0 {
				return 1 << 30
			}
			return n
		}
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].disc != rows[j].disc {
				return rows[i].disc < rows[j].disc
			}
			return order(rows[i].t.Number) < order(rows[j].t.Number)
		})
	}
	c.Tracks = make([]player.Track, 0, len(rows))
	for i, r := range rows {
		if kind == "playlist" {
			r.t.Number = i + 1
		}
		c.Tracks = append(c.Tracks, r.t)
	}
	if url, err := m.Artwork(ctx, id); err == nil {
		c.ArtworkURL = url
	}
	return c, nil
}
