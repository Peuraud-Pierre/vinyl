package catalog

import (
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vinyl/internal/player"
)

func writeSolid(t *testing.T, path string, c color.Color, size int) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			img.Set(x, y, c)
		}
	}
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
}

// Redimensionnement factice : une image unie de la couleur du coin de la source, à la bonne taille.
func fakeResize(t *testing.T) func(src, dst string, size int) error {
	return func(src, dst string, size int) error {
		f, err := os.Open(src)
		if err != nil {
			return err
		}
		defer f.Close()
		img, _, err := image.Decode(f)
		if err != nil {
			return err
		}
		writeSolid(t, dst, img.At(0, 0), size)
		return nil
	}
}

// fileRunner simule l'export AppleScript : il écrit n pochettes unies au préfixe reçu (argv 2).
type fileRunner struct {
	t      *testing.T
	colors []color.Color
	calls  int
	args   []string
}

func (r *fileRunner) Run(_ context.Context, _ string, args ...string) (string, error) {
	r.calls++
	r.args = args
	for i, c := range r.colors {
		writeSolid(r.t, args[1]+string(rune('1'+i))+".img", c, 40)
	}
	return string(rune('0' + len(r.colors))), nil
}

func near(a, b color.Color) bool {
	r1, g1, b1, _ := a.RGBA()
	r2, g2, b2, _ := b.RGBA()
	d := func(x, y uint32) bool { return x>>8 < y>>8+24 && y>>8 < x>>8+24 }
	return d(r1, r2) && d(g1, g2) && d(b1, b2)
}

var (
	red    = color.RGBA{220, 30, 30, 255}
	green  = color.RGBA{30, 200, 40, 255}
	blue   = color.RGBA{30, 40, 210, 255}
	yellow = color.RGBA{230, 220, 30, 255}
)

func decodeFile(t *testing.T, path string) image.Image {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	return img
}

// Comme dans Musique : une playlist sans image propre affiche les pochettes de ses quatre premiers
// albums différents, en 2 × 2 (haut gauche, haut droite, bas gauche, bas droite).
func TestPlaylistArtworkIsAMosaicOfFourAlbums(t *testing.T) {
	dir := t.TempDir()
	r := &fileRunner{t: t, colors: []color.Color{red, green, blue, yellow}}
	m := NewMusicLibrary(r, nil, dir)
	m.resize = fakeResize(t)
	u, err := m.Artwork(context.Background(), "playlist:P1")
	if err != nil || !strings.HasPrefix(u, "/artwork/") {
		t.Fatalf("url=%q err=%v", u, err)
	}
	img := decodeFile(t, filepath.Join(dir, strings.TrimPrefix(u, "/artwork/")))
	b := img.Bounds()
	if b.Dx() != 2*mosaicTile || b.Dy() != 2*mosaicTile {
		t.Fatalf("taille %v, attendu %d px de côté", b, 2*mosaicTile)
	}
	q := mosaicTile / 2
	for _, c := range []struct {
		x, y int
		want color.Color
		name string
	}{{q, q, red, "haut gauche"}, {3 * q, q, green, "haut droite"}, {q, 3 * q, blue, "bas gauche"}, {3 * q, 3 * q, yellow, "bas droite"}} {
		if got := img.At(c.x, c.y); !near(got, c.want) {
			t.Errorf("%s : %v, attendu %v", c.name, got, c.want)
		}
	}
	// Les pochettes intermédiaires ne traînent pas dans le dossier.
	if ents, _ := os.ReadDir(dir); len(ents) != 1 {
		t.Errorf("fichiers restants : %d", len(ents))
	}
}

// Moins de quatre albums : Musique montre simplement la pochette du premier.
func TestPlaylistArtworkWithFewAlbumsUsesTheFirstCover(t *testing.T) {
	dir := t.TempDir()
	m := NewMusicLibrary(&fileRunner{t: t, colors: []color.Color{green, blue}}, nil, dir)
	m.resize = fakeResize(t)
	u, err := m.Artwork(context.Background(), "playlist:P2")
	if err != nil || u == "" {
		t.Fatalf("url=%q err=%v", u, err)
	}
	img := decodeFile(t, filepath.Join(dir, strings.TrimPrefix(u, "/artwork/")))
	if !near(img.At(5, 5), green) {
		t.Errorf("pochette du premier album attendue, couleur %v", img.At(5, 5))
	}
}

// Les anciennes images de playlist (pochette du premier titre) déjà en cache ne doivent pas être reprises.
func TestPlaylistArtworkIgnoresOldCache(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, player.ArtworkFileName("item:playlist:P3")), []byte("ancienne"), 0o644)
	r := &fileRunner{t: t, colors: []color.Color{red}}
	m := NewMusicLibrary(r, nil, dir)
	m.resize = fakeResize(t)
	if _, err := m.Artwork(context.Background(), "playlist:P3"); err != nil || r.calls != 1 {
		t.Fatalf("la mosaïque doit être recalculée (appels %d, err %v)", r.calls, err)
	}
	if r.args[0] != "P3" {
		t.Errorf("identifiant de playlist transmis : %q", r.args[0])
	}
}

func TestPlaylistWithoutAnyCover(t *testing.T) {
	r := &fileRunner{t: t}
	m := NewMusicLibrary(r, nil, t.TempDir())
	for range 2 {
		if u, _ := m.Artwork(context.Background(), "playlist:P4"); u != "" {
			t.Fatalf("aucune pochette attendue : %q", u)
		}
	}
	if r.calls != 1 {
		t.Errorf("absence mémorisée attendue : %d appels", r.calls)
	}
}

// « Favourite Songs » : comme dans Musique, une étoile blanche sur un dégradé rose-rouge.
type favRunner struct{ calls int }

func (r *favRunner) Run(context.Context, string, ...string) (string, error) {
	r.calls++
	return "FAV", nil
}

func TestFavouriteSongsGetsTheStar(t *testing.T) {
	dir := t.TempDir()
	r := &favRunner{}
	m := NewMusicLibrary(r, nil, dir)
	u, err := m.Artwork(context.Background(), "playlist:F")
	if err != nil || u == "" {
		t.Fatalf("url=%q err=%v", u, err)
	}
	img := decodeFile(t, filepath.Join(dir, strings.TrimPrefix(u, "/artwork/")))
	s := img.Bounds().Dx()
	if s != 2*mosaicTile || img.Bounds().Dy() != s {
		t.Fatalf("taille %v", img.Bounds())
	}
	white := color.RGBA{255, 255, 255, 255}
	if c := img.At(s/2, s/2); !near(c, white) {
		t.Errorf("centre (étoile) : %v, attendu blanc", c)
	}
	for _, p := range []image.Point{{8, 8}, {s - 8, s - 8}} {
		r8, g8, b8, _ := img.At(p.X, p.Y).RGBA()
		if !(r8>>8 > 200 && g8>>8 < 120 && b8>>8 < 150) {
			t.Errorf("coin %v : %v, attendu rose-rouge", p, img.At(p.X, p.Y))
		}
	}
}

func TestPlaylistScriptRecognisesFavouriteSongs(t *testing.T) {
	for _, name := range []string{"Favourite Songs", "Favorite Songs", "Morceaux favoris"} {
		if !strings.Contains(playlistArtworkScript, `"`+name+`"`) {
			t.Errorf("nom %q non reconnu", name)
		}
	}
	if !strings.Contains(playlistArtworkScript, "smart of p") {
		t.Error("seule une playlist intelligente doit être prise pour « Favourite Songs »")
	}
}
