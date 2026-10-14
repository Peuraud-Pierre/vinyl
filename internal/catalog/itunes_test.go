package catalog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestITunesArtist(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		switch {
		case r.URL.Path == "/search" && q.Get("entity") == "musicArtist" && q.Get("term") == "Daft Punk" && q.Get("country") == "FR":
			w.Write([]byte(`{"results":[{"artistId":5468295,"artistName":"Daft Punk","artistLinkUrl":"https://music.apple.com/fr/artist/daft-punk/5468295"}]}`))
		case r.URL.Path == "/lookup" && q.Get("entity") == "song":
			w.Write([]byte(`{"results":[{"wrapperType":"artist"},{"wrapperType":"track","trackId":1,"trackName":"One More Time","collectionName":"Discovery","artworkUrl100":"https://x/100x100bb.jpg","trackViewUrl":"https://music.apple.com/fr/album/x?i=1"}]}`))
		case r.URL.Path == "/lookup" && q.Get("entity") == "album":
			w.Write([]byte(`{"results":[{"wrapperType":"artist"},{"wrapperType":"collection","collectionId":2,"collectionName":"Discovery","releaseDate":"2001-03-12T08:00:00Z","artworkUrl100":"https://x/100x100bb.jpg","collectionViewUrl":"https://music.apple.com/fr/album/2"}]}`))
		default:
			http.Error(w, "inattendu "+r.URL.String(), 400)
		}
	}))
	defer srv.Close()
	c := &ITunes{BaseURL: srv.URL, HTTP: srv.Client()}
	p, err := c.Artist(context.Background(), "Daft Punk")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "Daft Punk" || len(p.TopTracks) != 1 || len(p.Albums) != 1 {
		t.Fatalf("got %+v", p)
	}
	if p.TopTracks[0].ArtworkURL != "https://x/600x600bb.jpg" || p.Albums[0].Subtitle != "2001" {
		t.Errorf("pochette HD / année : %+v %+v", p.TopTracks[0], p.Albums[0])
	}
	if p.ImageURL != "https://x/600x600bb.jpg" {
		t.Errorf("image d'en-tête = %s", p.ImageURL)
	}
}

func TestITunesArtistNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(`{"results":[]}`)) }))
	defer srv.Close()
	_, err := (&ITunes{BaseURL: srv.URL, HTTP: srv.Client()}).Artist(context.Background(), "Inconnu")
	if err == nil {
		t.Error("artiste introuvable → erreur attendue")
	}
}

func TestITunesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "boom", 503) }))
	defer srv.Close()
	if _, err := (&ITunes{BaseURL: srv.URL, HTTP: srv.Client()}).Artist(context.Background(), "X"); err == nil {
		t.Error("HTTP 503 → erreur attendue")
	}
}
