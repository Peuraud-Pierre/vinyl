package catalog

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"math"
	"os"
	"strconv"
	"strings"

	"vinyl/internal/player"
)

// mosaicTile est la taille d'une case de la mosaïque des playlists (image finale : 2 × 2 cases).
const mosaicTile = 300

// argv : persistent ID de la playlist, préfixe des fichiers. Comme Musique pour une playlist sans
// image propre : pochettes des quatre premiers albums différents, dans l'ordre de la playlist.
// Écrit <préfixe>1.img … <préfixe>4.img et renvoie leur nombre ; « FAV » pour « Favourite Songs »
// (playlist intelligente de Musique, reconnue à son nom), qui a son image à elle : l'étoile.
const playlistArtworkScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	set found to 0
	tell application id "com.apple.Music"
		set p to (first playlist whose persistent ID is (item 1 of argv))
		set fav to false
		try -- les playlists abonnées n'ont pas de propriété « smart »
			if smart of p and ({"Favourite Songs", "Favorite Songs", "Morceaux favoris", "Titres favoris", "Lieblingssongs", "Canciones favoritas", "Brani preferiti"} contains (name of p)) then set fav to true
		end try
		if fav then return "FAV"
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
	switch out {
	case "NOTRUNNING":
		return false, player.ErrNotRunning
	case "FAV":
		return true, writeJPEG(favouriteStar(2*mosaicTile), path)
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
	return writeJPEG(out, path)
}

func writeJPEG(img image.Image, path string) error {
	tmp := path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 90}); err != nil {
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

// favouriteStar dessine l'image de « Favourite Songs » : étoile en dégradé rose-rouge (haut gauche →
// bas droite) au centre d'un fond blanc. Bords lissés par suréchantillonnage 4 × 4.
func favouriteStar(size int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	from, to := [3]float64{255, 94, 122}, [3]float64{247, 33, 60}
	cx, cy := float64(size)/2, float64(size)/2
	outer := 0.30 * float64(size)
	inner := outer * 0.42
	var star [10][2]float64
	for i := range star {
		r := outer
		if i%2 == 1 {
			r = inner
		}
		a := -math.Pi/2 + float64(i)*math.Pi/5
		star[i] = [2]float64{cx + r*math.Cos(a), cy + r*math.Sin(a)}
	}
	inside := func(x, y float64) bool {
		in := false
		for i, j := 0, len(star)-1; i < len(star); j, i = i, i+1 {
			xi, yi, xj, yj := star[i][0], star[i][1], star[j][0], star[j][1]
			if (yi > y) != (yj > y) && x < (xj-xi)*(y-yi)/(yj-yi)+xi {
				in = !in
			}
		}
		return in
	}
	const ss = 4
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			t := float64(x+y) / float64(2*(size-1))
			cover := 0.0
			if math.Abs(float64(x)-cx) <= outer+1 && math.Abs(float64(y)-cy) <= outer+1 {
				for sy := 0; sy < ss; sy++ {
					for sx := 0; sx < ss; sx++ {
						if inside(float64(x)+(float64(sx)+0.5)/ss, float64(y)+(float64(sy)+0.5)/ss) {
							cover++
						}
					}
				}
				cover /= ss * ss
			}
			var c [3]uint8
			for k := range c {
				bg := from[k] + (to[k]-from[k])*t        // teinte du dégradé à cet endroit
				c[k] = uint8(255 + (bg-255)*cover + 0.5) // blanc hors de l'étoile, dégradé dedans
			}
			img.SetRGBA(x, y, color.RGBA{c[0], c[1], c[2], 255})
		}
	}
	return img
}
