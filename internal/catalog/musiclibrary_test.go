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

func rec(fields ...string) string { return strings.Join(fields, player.Sep) + rs }

func TestMusicAlbumTracksSortedByDiscThenNumber(t *testing.T) {
	// id, disque, piste, titre, artiste, durée, aimé, genre, année
	out := columns("",
		[]string{"C", "2", "1", "Disque 2 piste 1", "Nirvana", "120,5", "false", "Alternative", "1993"},
		[]string{"B", "1", "2", `Heart-Shaped "Box"`, "Nirvana", "281", "true", "Alternative", "1993"},
		[]string{"Z", "1", "0", "Sans numéro", "Nirvana", "60", "false", "Alternative", "1993"},
		[]string{"A", "1", "1", "Serve the Servants", "Nirvana", "217", "false", "Alternative", "1993"})
	r := &fakeRunner{out: []string{out}}
	c, err := NewMusicLibrary(r, nil, t.TempDir()).Tracks(context.Background(), "album:In Utero"+player.Sep+"Nirvana")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, tr := range c.Tracks {
		ids = append(ids, tr.ID)
	}
	if strings.Join(ids, ",") != "A,B,Z,C" {
		t.Errorf("ordre = %v (disque, puis piste, sans numéro en fin de disque)", ids)
	}
	if c.Title != "In Utero" || c.Artist != "Nirvana" || c.Genre != "Alternative" || c.Year != "1993" || c.Kind != "album" {
		t.Errorf("en-tête = %+v", c)
	}
	if !c.Tracks[1].Loved || c.Tracks[1].DurationMs != 281000 || c.Tracks[3].DurationMs != 120500 {
		t.Errorf("titres = %+v", c.Tracks)
	}
	if got := r.calls[0][1:]; strings.Join(got, "|") != "In Utero|Nirvana" {
		t.Errorf("argv = %q", got)
	}
}

func TestMusicPlaylistTracksKeepOrderAndName(t *testing.T) {
	out := columns("Ma playlist",
		[]string{"Y", "0", "7", "Deux", "B", "200", "false", "", "0"},
		[]string{"X", "0", "3", "Un", "A", "100", "false", "", "0"})
	c, err := NewMusicLibrary(&fakeRunner{out: []string{out}}, nil, t.TempDir()).Tracks(context.Background(), "playlist:P1")
	if err != nil {
		t.Fatal(err)
	}
	if c.Title != "Ma playlist" || c.Kind != "playlist" || len(c.Tracks) != 2 || c.Tracks[0].ID != "Y" || c.Tracks[0].Number != 1 || c.Year != "" {
		t.Errorf("got %+v", c)
	}
}

func TestMusicTracksNotRunningAndBadID(t *testing.T) {
	m := NewMusicLibrary(&fakeRunner{out: []string{"NOTRUNNING"}}, nil, t.TempDir())
	if _, err := m.Tracks(context.Background(), "album:X"+player.Sep+"Y"); err != player.ErrNotRunning {
		t.Errorf("got %v", err)
	}
	if _, err := m.Tracks(context.Background(), "inconnu"); err == nil {
		t.Error("ID invalide → erreur")
	}
}

// Les playlists ajoutées depuis Apple Music (ex. « zons ») sont des « subscription playlist »,
// pas des « user playlist » : elles doivent être listées, lues et lancées comme les autres.
func TestApplemusicSubscriptionPlaylistsAreIncluded(t *testing.T) {
	if !strings.Contains(playlistsScript, "every subscription playlist") {
		t.Error("la liste des playlists doit inclure les playlists Apple Music ajoutées (subscription playlist)")
	}
	for name, script := range map[string]string{"titres": playlistTracksScript, "pochette": itemArtworkScript} {
		if strings.Contains(script, "first user playlist whose persistent ID") {
			t.Errorf("%s : la recherche par ID doit couvrir toutes les playlists (first playlist whose …)", name)
		}
	}
}

// Un album sans « album artist » est listé sous l'artiste du titre : la recherche de ses titres
// doit d'abord essayer cet artiste avant de se rabattre sur le seul nom d'album (sinon les
// « Greatest Hits » de plusieurs artistes se mélangent).
func TestAlbumTracksScriptTriesTrackArtistBeforeAlbumOnly(t *testing.T) {
	i := strings.Index(albumTracksScript, "whose album is (item 1 of argv) and artist is (item 2 of argv)")
	j := strings.LastIndex(albumTracksScript, "whose album is (item 1 of argv))")
	if i < 0 || j < 0 || i > j {
		t.Error("ordre attendu : album + album artist, puis album + artist, puis album seul")
	}
}

// columns construit la sortie « en colonnes » des scripts de titres : en-tête, puis 9 colonnes
// (id, disque, piste, titre, artiste, durée, aimé, genre, année) séparées par \x1d.
func columns(header string, rows ...[]string) string {
	cols := make([][]string, 9)
	for _, r := range rows {
		for i := range cols {
			cols[i] = append(cols[i], r[i])
		}
	}
	parts := []string{header}
	for _, c := range cols {
		parts = append(parts, strings.Join(c, rs))
	}
	return strings.Join(parts, "\x1d")
}

// Les grosses playlists doivent être lues en bloc (une requête par colonne), pas titre par titre.
func TestTracksScriptsReadColumnsInBulk(t *testing.T) {
	for name, script := range map[string]string{"album": albumTracksScript, "playlist": playlistTracksScript} {
		if strings.Contains(script, "repeat with t in") {
			t.Errorf("%s : lecture titre par titre (lent sur les grosses playlists)", name)
		}
	}
}
