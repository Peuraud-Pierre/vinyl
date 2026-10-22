package lyrics

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
)

// Paroles inventées pour les tests.
func TestParseLRC(t *testing.T) {
	got := ParseLRC("[ar:Personne]\n[00:01.5] Première ligne\r\n[00:10.25][01:02.000] Refrain\n[00:05.00]\nsans horodatage\n")
	want := []Line{{1500, "Première ligne"}, {5000, ""}, {10250, "Refrain"}, {62000, "Refrain"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v\nwant %+v", got, want)
	}
}

func TestFindExactThenSearch(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path+"?"+r.URL.RawQuery)
		if r.Header.Get("User-Agent") == "" {
			t.Error("User-Agent manquant")
		}
		switch r.URL.Path {
		case "/api/get":
			http.NotFound(w, r)
		case "/api/search":
			w.Write([]byte(`[
				{"duration": 300, "syncedLyrics": "[00:01.00] trop long"},
				{"duration": 181, "plainLyrics": "texte seul"},
				{"duration": 183, "syncedLyrics": "[00:02.00] la bonne", "plainLyrics": "la bonne"}
			]`))
		}
	}))
	defer srv.Close()
	c := &LRCLib{BaseURL: srv.URL, HTTP: srv.Client(), UserAgent: "test"}
	l, err := c.Find(context.Background(), "Titre", "Artiste", "Album", 181400)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Synced) != 1 || l.Synced[0].Text != "la bonne" {
		t.Errorf("got %+v", l)
	}
	if len(paths) != 2 || paths[0] != "/api/get?album_name=Album&artist_name=Artiste&duration=181&track_name=Titre" {
		t.Errorf("requêtes %v", paths)
	}
}

func TestFindNothing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/search" {
			w.Write([]byte(`[]`))
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()
	c := &LRCLib{BaseURL: srv.URL, HTTP: srv.Client(), UserAgent: "test"}
	l, err := c.Find(context.Background(), "Titre", "Artiste", "", 0)
	if err != nil || !l.Empty() {
		t.Errorf("got %+v, %v", l, err)
	}
}
