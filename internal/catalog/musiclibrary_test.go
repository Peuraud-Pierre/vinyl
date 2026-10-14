package catalog

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"vinyl/internal/player"
)

type fakeRunner struct {
	out   []string
	calls [][]string
}

func (f *fakeRunner) Run(_ context.Context, script string, args ...string) (string, error) {
	i := len(f.calls)
	f.calls = append(f.calls, append([]string{script}, args...))
	if i < len(f.out) {
		return f.out[i], nil
	}
	return "", nil
}

const rs = "\x1e" // séparateur d'enregistrements

func TestMusicPlaylists(t *testing.T) {
	r := &fakeRunner{out: []string{"Chill" + player.Sep + "P1" + player.Sep + "12" + rs + "Rock" + player.Sep + "P2" + player.Sep + "3"}}
	p, err := NewMusicLibrary(r, nil, t.TempDir()).Library(context.Background(), "playlists", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != 2 || p.Items[0] != (Item{ID: "playlist:P1", Kind: "playlist", Title: "Chill", Subtitle: "12 titres"}) || p.Next != -1 {
		t.Errorf("got %+v", p)
	}
}

func TestMusicAlbumsDedupSortAndPaginate(t *testing.T) {
	var b strings.Builder
	// 60 albums distincts, chacun en double, ordre inverse
	for i := 59; i >= 0; i-- {
		for j := 0; j < 2; j++ {
			b.WriteString("Album " + string(rune('A'+i/26)) + string(rune('a'+i%26)) + player.Sep + "Artiste" + rs)
		}
	}
	r := &fakeRunner{out: []string{b.String(), b.String()}}
	m := NewMusicLibrary(r, nil, t.TempDir())
	p, err := m.Library(context.Background(), "albums", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Items) != PageSize || p.Next != PageSize || p.Items[0].Title != "Album Aa" {
		t.Fatalf("page 1 : %d éléments, next=%d, premier=%q", len(p.Items), p.Next, p.Items[0].Title)
	}
	if p.Items[0].ID != "album:Album Aa"+player.Sep+"Artiste" {
		t.Errorf("ID = %q", p.Items[0].ID)
	}
	p2, _ := m.Library(context.Background(), "albums", PageSize)
	if len(p2.Items) != 10 || p2.Next != -1 {
		t.Errorf("page 2 : %d éléments, next=%d", len(p2.Items), p2.Next)
	}
}

func TestMusicLikesUnsupportedAndNotRunning(t *testing.T) {
	m := NewMusicLibrary(&fakeRunner{out: []string{"NOTRUNNING"}}, nil, t.TempDir())
	if _, err := m.Library(context.Background(), "likes", 0); err != ErrUnsupported {
		t.Errorf("likes : %v", err)
	}
	if _, err := m.Library(context.Background(), "playlists", 0); err != player.ErrNotRunning {
		t.Errorf("Musique fermé : %v", err)
	}
}

type slowRunner struct {
	mu      sync.Mutex
	running int
	maxSeen int
	calls   int
	out     string
	hold    time.Duration
}

func (r *slowRunner) Run(context.Context, string, ...string) (string, error) {
	r.mu.Lock()
	r.running++
	r.calls++
	r.maxSeen = max(r.maxSeen, r.running)
	r.mu.Unlock()
	time.Sleep(r.hold)
	r.mu.Lock()
	r.running--
	r.mu.Unlock()
	return r.out, nil
}

func TestArtworkExportsAreLimitedToTwoAtOnce(t *testing.T) {
	r := &slowRunner{out: "NONE", hold: 20 * time.Millisecond}
	m := NewMusicLibrary(r, nil, t.TempDir())
	var wg sync.WaitGroup
	for i := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			m.Artwork(context.Background(), "album:A"+strconv.Itoa(i)+player.Sep+"X")
		}()
	}
	wg.Wait()
	if r.maxSeen > 2 {
		t.Errorf("au plus 2 exports simultanés vers Musique, vu %d", r.maxSeen)
	}
}

func TestArtworkRemembersMissingCover(t *testing.T) {
	r := &slowRunner{out: "NONE"}
	m := NewMusicLibrary(r, nil, t.TempDir())
	for range 3 {
		if url, _ := m.Artwork(context.Background(), "album:A"+player.Sep+"X"); url != "" {
			t.Fatalf("pas de pochette attendu, obtenu %q", url)
		}
	}
	if r.calls != 1 {
		t.Errorf("une absence de pochette doit être mémorisée : %d appels", r.calls)
	}
}
