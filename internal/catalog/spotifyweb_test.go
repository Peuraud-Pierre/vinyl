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
