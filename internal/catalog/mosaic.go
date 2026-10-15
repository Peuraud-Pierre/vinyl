package catalog

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"os"
	"strconv"
	"strings"

	"vinyl/internal/player"
)

// mosaicTile est la taille d'une case de la mosaïque des playlists (image finale : 2 × 2 cases).
const mosaicTile = 300

// argv : persistent ID de la playlist, préfixe des fichiers. Comme Musique pour une playlist sans
// image propre : pochettes des quatre premiers albums différents, dans l'ordre de la playlist.
// Écrit <préfixe>1.img … <préfixe>4.img et renvoie leur nombre.
const playlistArtworkScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	set found to 0
	tell application id "com.apple.Music"
		set p to (first playlist whose persistent ID is (item 1 of argv))
		set ts to a reference to (every track of p)
		set n to count of ts
		if n is 0 then return "0"
		set als to album of ts
		if n > 400 then set n to 400
		set seenAlbums to {}
		repeat with i from 1 to n
			set a to item i of als
			if seenAlbums does not contain {a} then
				set end of seenAlbums to a
				set t to item i of ts
				if (count of artworks of t) > 0 then
					set img to raw data of artwork 1 of t
					set found to found + 1
					my saveImage(img, (item 2 of argv) & found & ".img")
					if found is 4 then exit repeat
				end if
			end if
		end repeat
	end tell
	return found as text
end run

on saveImage(img, dest)
	set f to open for access (POSIX file dest) with write permission
	try
		set eof f to 0
		write img to f
	end try
	close access f
end saveImage`

// playlistArtwork exporte les pochettes et compose l'image de la playlist dans path.
// Renvoie false si la playlist n'a aucune pochette.
func (m *MusicLibrary) playlistArtwork(ctx context.Context, id, path string) (bool, error) {
	prefix := path + ".part"
	out, err := m.run.Run(ctx, playlistArtworkScript, id, prefix)
	if err != nil {
		return false, err
	}
	if out == "NOTRUNNING" {
		return false, player.ErrNotRunning
	}
	n, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		return false, fmt.Errorf("musique : réponse inattendue %q", out)
	}
	srcs := make([]string, n)
	for i := range srcs {
		srcs[i] = prefix + strconv.Itoa(i+1) + ".img"
		defer os.Remove(srcs[i])
	}
	switch {
	case n == 0:
		return false, nil
	case n < 4:
		return true, os.Rename(srcs[0], path)
	default:
		return true, m.mosaic(srcs[:4], path)
	}
}

// mosaic assemble quatre pochettes en 2 × 2 (JPEG).
func (m *MusicLibrary) mosaic(srcs []string, path string) error {
	out := image.NewRGBA(image.Rect(0, 0, 2*mosaicTile, 2*mosaicTile))
	for i, src := range srcs {
		tile := src + ".tile.jpg"
		if err := m.resize(src, tile, mosaicTile); err != nil {
			return err
		}
		img, err := decodeImage(tile)
		os.Remove(tile)
		if err != nil {
			return err
		}
		at := image.Pt((i%2)*mosaicTile, (i/2)*mosaicTile)
		draw.Draw(out, image.Rectangle{Min: at, Max: at.Add(image.Pt(mosaicTile, mosaicTile))}, img, img.Bounds().Min, draw.Src)
	}
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := jpeg.Encode(f, out, &jpeg.Options{Quality: 90}); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func decodeImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}
