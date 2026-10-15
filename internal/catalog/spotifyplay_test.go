package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestSpotifyPlayStartsTrackInContext(t *testing.T) {
	var calls []string
	var body map[string]any
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		if r.URL.Path == "/me/player/play" {
			json.NewDecoder(r.Body).Decode(&body)
		}
		w.WriteHeader(204)
	})
	if err := c.Play(context.Background(), "spotify:album:A", "spotify:track:T", true); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "PUT /me/player/shuffle?state=true" || calls[1] != "PUT /me/player/play?" {
		t.Errorf("appels %v", calls)
	}
	if body["context_uri"] != "spotify:album:A" || body["offset"].(map[string]any)["uri"] != "spotify:track:T" {
		t.Errorf("corps %v", body)
	}
}

// Un titre seul : lecture de ce titre.
func TestSpotifyPlayTrack(t *testing.T) {
	var body map[string]any
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&body)
		w.WriteHeader(204)
	})
	if err := c.PlayTrack(context.Background(), "spotify:track:T"); err != nil {
		t.Fatal(err)
	}
	if uris, _ := body["uris"].([]any); len(uris) != 1 || uris[0] != "spotify:track:T" {
		t.Errorf("corps %v", body)
	}
}

// Pas Premium (403) ou aucun appareil (404) : erreur reconnaissable pour le repli.
func TestSpotifyPlayRefused(t *testing.T) {
	for _, code := range []int{403, 404} {
		c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(code) })
		if err := c.Play(context.Background(), "spotify:album:A", "", false); !errors.Is(err, ErrPlaybackRefused) {
			t.Errorf("%d : %v", code, err)
		}
	}
}
