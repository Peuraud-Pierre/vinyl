package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ErrPlaybackRefused : Spotify refuse de piloter la lecture (compte non Premium, aucun appareil actif).
var ErrPlaybackRefused = errors.New("Spotify refuse de lancer la lecture (Premium et app ouverte nécessaires)")

func (c *SpotifyWeb) put(ctx context.Context, path string, q url.Values, body any) error {
	tok, err := c.Token(ctx)
	if err != nil {
		return err
	}
	u := c.BaseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, err := http.NewRequestWithContext(ctx, "PUT", u, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	switch resp.StatusCode {
	case 200, 202, 204:
		return nil
	case 403, 404:
		return ErrPlaybackRefused
	case 401:
		return fmt.Errorf("session Spotify expirée, reconnecte-toi")
	}
	return fmt.Errorf("spotify: HTTP %d", resp.StatusCode)
}

// Play lance un album / une playlist (contextURI) à partir d'un titre (trackURI, vide = début).
func (c *SpotifyWeb) Play(ctx context.Context, contextURI, trackURI string, shuffle bool) error {
	if err := c.put(ctx, "/me/player/shuffle", url.Values{"state": {strconv.FormatBool(shuffle)}}, nil); err != nil {
		return err
	}
	body := map[string]any{"context_uri": contextURI}
	if trackURI != "" {
		body["offset"] = map[string]any{"uri": trackURI}
	}
	return c.put(ctx, "/me/player/play", nil, body)
}

// PlayTrack lance un titre seul.
func (c *SpotifyWeb) PlayTrack(ctx context.Context, trackURI string) error {
	return c.put(ctx, "/me/player/play", nil, map[string]any{"uris": []string{trackURI}})
}
