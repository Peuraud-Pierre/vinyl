package catalog

import (
	"context"
	"os"
	"path/filepath"
	"sort"
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
			set ts to tracks of (first user playlist whose persistent ID is (item 2 of argv))
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
	out, err := m.run.Run(ctx, script)
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
