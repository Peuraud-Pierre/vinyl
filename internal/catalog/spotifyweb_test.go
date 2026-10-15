package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newSpotifyTest(t *testing.T, h http.HandlerFunc) *SpotifyWeb {
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return &SpotifyWeb{BaseURL: srv.URL, HTTP: srv.Client(), Token: func(context.Context) (string, error) { return "TOK", nil }}
}

func TestSpotifyLikes(t *testing.T) {
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer TOK" || r.URL.Path != "/me/tracks" || r.URL.Query().Get("offset") != "50" || r.URL.Query().Get("limit") != "50" {
			http.Error(w, r.URL.String(), 400)
			return
		}
		w.Write([]byte(`{"items":[{"track":{"uri":"spotify:track:1","name":"T","artists":[{"name":"A"},{"name":"B"}],"album":{"images":[{"url":"https://img/640"}]}}}],"next":null}`))
	})
	p, err := c.Library(context.Background(), "likes", 50)
	if err != nil {
		t.Fatal(err)
	}
	want := Item{ID: "spotify:track:1", Kind: "track", Title: "T", Subtitle: "A, B", ArtworkURL: "https://img/640"}
	if len(p.Items) != 1 || p.Items[0] != want || p.Next != -1 {
		t.Errorf("got %+v", p)
	}
}

func TestSpotifyPlaylistsNextOffset(t *testing.T) {
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"items":[{"uri":"spotify:playlist:9","name":"Mix","owner":{"display_name":"moi"},"images":[]}],"next":"https://api.spotify.com/v1/me/playlists?offset=50"}`))
	})
	p, _ := c.Library(context.Background(), "playlists", 0)
	if p.Next != 50 || p.Items[0].ArtworkURL != "" || p.Items[0].Subtitle != "moi" {
		t.Errorf("got %+v", p)
	}
}

func TestSpotifyArtistFromTrackID(t *testing.T) {
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/tracks/42":
			w.Write([]byte(`{"artists":[{"id":"AR"}]}`))
		case "/artists/AR":
			w.Write([]byte(`{"name":"Daft Punk","uri":"spotify:artist:AR","external_urls":{"spotify":"https://open.spotify.com/artist/AR"},"images":[{"url":"https://img/a"}]}`))
		case "/artists/AR/top-tracks":
			w.Write([]byte(`{"tracks":[{"uri":"spotify:track:1","name":"One More Time","album":{"name":"Discovery","images":[{"url":"https://img/d"}]}}]}`))
		case "/artists/AR/albums":
			w.Write([]byte(`{"items":[{"uri":"spotify:album:7","name":"Discovery","release_date":"2001-03-12","images":[{"url":"https://img/d"}]}]}`))
		default:
			http.Error(w, r.URL.Path, 404)
		}
	})
	p, err := c.Artist(context.Background(), "Daft Punk", "spotify:track:42")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Daft Punk" || p.ImageURL != "https://img/a" || len(p.TopTracks) != 1 || p.Albums[0].Subtitle != "2001" || p.URL != "spotify:artist:AR" {
		t.Errorf("got %+v", p)
	}
}

func TestSpotifyRateLimitedError(t *testing.T) {
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "7")
		w.WriteHeader(429)
	})
	_, err := c.Library(context.Background(), "playlists", 0)
	if err == nil || err.Error() != "Spotify limite les requêtes, réessaie dans 7 s" {
		t.Errorf("got %v", err)
	}
}

func TestSpotifyAlbumTracksWithHearts(t *testing.T) {
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/albums/AL":
			w.Write([]byte(`{"name":"In Utero","release_date":"1993-09-21","genres":[],"artists":[{"name":"Nirvana"}],"images":[{"url":"https://img/u"}],
				"tracks":{"items":[{"uri":"spotify:track:1","name":"Serve the Servants","track_number":1,"duration_ms":217000,"artists":[{"name":"Nirvana"}]},
				{"uri":"spotify:track:2","name":"Scentless Apprentice","track_number":2,"duration_ms":228000,"artists":[{"name":"Nirvana"}]}]}}`))
		case "/me/tracks/contains":
			if r.URL.Query().Get("ids") != "1,2" {
				http.Error(w, "ids", 400)
				return
			}
			w.Write([]byte(`[false,true]`))
		default:
			http.Error(w, r.URL.Path, 404)
		}
	})
	col, err := c.Tracks(context.Background(), "spotify:album:AL")
	if err != nil {
		t.Fatal(err)
	}
	if col.Title != "In Utero" || col.Artist != "Nirvana" || col.Year != "1993" || col.ArtworkURL != "https://img/u" || len(col.Tracks) != 2 || !col.Tracks[1].Loved || col.Tracks[0].Loved {
		t.Errorf("got %+v", col)
	}
}

func TestSpotifyPlaylistTracksPaginated(t *testing.T) {
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/playlists/PL":
			w.Write([]byte(`{"name":"Mix","owner":{"display_name":"moi"},"images":[{"url":"https://img/p"}]}`))
		case r.URL.Path == "/playlists/PL/tracks" && r.URL.Query().Get("offset") == "0":
			w.Write([]byte(`{"items":[{"track":{"uri":"spotify:track:1","name":"A","duration_ms":1000,"artists":[{"name":"X"}]}}],"next":"more"}`))
		case r.URL.Path == "/playlists/PL/tracks" && r.URL.Query().Get("offset") == "100":
			w.Write([]byte(`{"items":[{"track":null},{"track":{"uri":"spotify:track:2","name":"B","duration_ms":2000,"artists":[{"name":"Y"}]}}],"next":null}`))
		case r.URL.Path == "/me/tracks/contains":
			w.Write([]byte(`[false,false]`))
		default:
			http.Error(w, r.URL.String(), 404)
		}
	})
	col, err := c.Tracks(context.Background(), "spotify:playlist:PL")
	if err != nil {
		t.Fatal(err)
	}
	if col.Title != "Mix" || col.Artist != "moi" || len(col.Tracks) != 2 || col.Tracks[1].Number != 2 || col.Tracks[1].Title != "B" {
		t.Errorf("got %+v", col)
	}
}

func TestSpotifyQueue(t *testing.T) {
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/player/queue" {
			http.Error(w, r.URL.Path, 404)
			return
		}
		w.Write([]byte(`{"currently_playing":{"uri":"spotify:track:0"},"queue":[{"uri":"spotify:track:1","name":"Heart Flutters","duration_ms":1000,"artists":[{"name":"Egofear"}],"album":{"name":"Colors"}}]}`))
	})
	q, err := c.Queue(context.Background())
	if err != nil || len(q) != 1 || q[0].Title != "Heart Flutters" || q[0].Artist != "Egofear" || q[0].Album != "Colors" {
		t.Errorf("got %+v %v", q, err)
	}
}

func TestShelfImagePicksTheLightestLargeEnough(t *testing.T) {
	imgs := []spImage{{URL: "big", Width: 640}, {URL: "mid", Width: 300}, {URL: "tiny", Width: 64}}
	if got := shelfImage(imgs); got != "mid" {
		t.Errorf("got %q", got)
	}
	if got := shelfImage([]spImage{{URL: "seule"}}); got != "seule" {
		t.Errorf("sans tailles : première image, got %q", got)
	}
	if shelfImage(nil) != "" {
		t.Error("aucune image → vide")
	}
}
