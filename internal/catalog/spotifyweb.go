package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// SpotifyWeb lit la bibliothèque et les pages artiste via la Spotify Web API.
type SpotifyWeb struct {
	BaseURL string
	HTTP    *http.Client
	Token   func(context.Context) (string, error)
}

func NewSpotifyWeb(token func(context.Context) (string, error)) *SpotifyWeb {
	return &SpotifyWeb{BaseURL: "https://api.spotify.com/v1", HTTP: &http.Client{Timeout: 10 * time.Second}, Token: token}
}

type spImage struct {
	URL string `json:"url"`
}

type spArtist struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type spAlbum struct {
	URI         string     `json:"uri"`
	Name        string     `json:"name"`
	ReleaseDate string     `json:"release_date"`
	Images      []spImage  `json:"images"`
	Artists     []spArtist `json:"artists"`
}

type spTrack struct {
	URI     string     `json:"uri"`
	Name    string     `json:"name"`
	Artists []spArtist `json:"artists"`
	Album   spAlbum    `json:"album"`
}

func firstImage(imgs []spImage) string {
	if len(imgs) == 0 {
		return ""
	}
	return imgs[0].URL
}

func artistNames(as []spArtist) string {
	names := make([]string, len(as))
	for i, a := range as {
		names[i] = a.Name
	}
	return strings.Join(names, ", ")
}

func (c *SpotifyWeb) get(ctx context.Context, path string, q url.Values, out any) error {
	tok, err := c.Token(ctx)
	if err != nil {
		return err
	}
	u := c.BaseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, "GET", u, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch {
	case resp.StatusCode == 429:
		return fmt.Errorf("Spotify limite les requêtes, réessaie dans %s s", resp.Header.Get("Retry-After"))
	case resp.StatusCode == 401:
		return fmt.Errorf("session Spotify expirée, reconnecte-toi")
	case resp.StatusCode != 200:
		return fmt.Errorf("spotify: HTTP %d", resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func nextOffset(next *string, offset int) int {
	if next == nil || *next == "" {
		return -1
	}
	return offset + PageSize
}

func (c *SpotifyWeb) Library(ctx context.Context, kind string, offset int) (Page, error) {
	q := url.Values{"limit": {strconv.Itoa(PageSize)}, "offset": {strconv.Itoa(offset)}}
	var page Page
	switch kind {
	case "playlists":
		var body struct {
			Items []struct {
				URI    string    `json:"uri"`
				Name   string    `json:"name"`
				Images []spImage `json:"images"`
				Owner  struct {
					DisplayName string `json:"display_name"`
				} `json:"owner"`
			} `json:"items"`
			Next *string `json:"next"`
		}
		if err := c.get(ctx, "/me/playlists", q, &body); err != nil {
			return Page{}, err
		}
		for _, p := range body.Items {
			page.Items = append(page.Items, Item{ID: p.URI, Kind: "playlist", Title: p.Name, Subtitle: p.Owner.DisplayName, ArtworkURL: firstImage(p.Images)})
		}
		page.Next = nextOffset(body.Next, offset)
	case "albums":
		var body struct {
			Items []struct {
				Album spAlbum `json:"album"`
			} `json:"items"`
			Next *string `json:"next"`
		}
		if err := c.get(ctx, "/me/albums", q, &body); err != nil {
			return Page{}, err
		}
		for _, a := range body.Items {
			page.Items = append(page.Items, Item{ID: a.Album.URI, Kind: "album", Title: a.Album.Name, Subtitle: artistNames(a.Album.Artists), ArtworkURL: firstImage(a.Album.Images)})
		}
		page.Next = nextOffset(body.Next, offset)
	case "likes":
		var body struct {
			Items []struct {
				Track spTrack `json:"track"`
			} `json:"items"`
			Next *string `json:"next"`
		}
		if err := c.get(ctx, "/me/tracks", q, &body); err != nil {
			return Page{}, err
		}
		for _, t := range body.Items {
			page.Items = append(page.Items, Item{ID: t.Track.URI, Kind: "track", Title: t.Track.Name, Subtitle: artistNames(t.Track.Artists), ArtworkURL: firstImage(t.Track.Album.Images)})
		}
		page.Next = nextOffset(body.Next, offset)
	case "artists":
		// /me/following pagine par curseur ; une seule page de 50 suffit pour un usage perso.
		if offset > 0 {
			return Page{Next: -1}, nil
		}
		var body struct {
			Artists struct {
				Items []struct {
					Name   string    `json:"name"`
					Images []spImage `json:"images"`
				} `json:"items"`
			} `json:"artists"`
		}
		if err := c.get(ctx, "/me/following", url.Values{"type": {"artist"}, "limit": {"50"}}, &body); err != nil {
			return Page{}, err
		}
		for _, a := range body.Artists.Items {
			page.Items = append(page.Items, Item{ID: a.Name, Kind: "artist", Title: a.Name, ArtworkURL: firstImage(a.Images)})
		}
		page.Next = -1
	default:
		return Page{}, ErrUnsupported
	}
	return page, nil
}

func (c *SpotifyWeb) artistID(ctx context.Context, name, trackID string) (string, error) {
	if id, ok := strings.CutPrefix(trackID, "spotify:track:"); ok {
		var t struct {
			Artists []spArtist `json:"artists"`
		}
		if err := c.get(ctx, "/tracks/"+id, nil, &t); err == nil && len(t.Artists) > 0 {
			return t.Artists[0].ID, nil
		}
	}
	var s struct {
		Artists struct {
			Items []spArtist `json:"items"`
		} `json:"artists"`
	}
	if err := c.get(ctx, "/search", url.Values{"type": {"artist"}, "q": {name}, "limit": {"1"}}, &s); err != nil {
		return "", err
	}
	if len(s.Artists.Items) == 0 {
		return "", fmt.Errorf("artiste introuvable : %s", name)
	}
	return s.Artists.Items[0].ID, nil
}

func (c *SpotifyWeb) Artist(ctx context.Context, name, trackID string) (ArtistPage, error) {
	id, err := c.artistID(ctx, name, trackID)
	if err != nil {
		return ArtistPage{}, err
	}
	var a struct {
		Name   string    `json:"name"`
		URI    string    `json:"uri"`
		Images []spImage `json:"images"`
	}
	if err := c.get(ctx, "/artists/"+id, nil, &a); err != nil {
		return ArtistPage{}, err
	}
	page := ArtistPage{Source: "spotify", Name: a.Name, ImageURL: firstImage(a.Images), URL: a.URI}

	var top struct {
		Tracks []spTrack `json:"tracks"`
	}
	if err := c.get(ctx, "/artists/"+id+"/top-tracks", url.Values{"market": {"FR"}}, &top); err != nil {
		return ArtistPage{}, err
	}
	for _, t := range top.Tracks {
		page.TopTracks = append(page.TopTracks, Item{ID: t.URI, Kind: "track", Title: t.Name, Subtitle: t.Album.Name, ArtworkURL: firstImage(t.Album.Images)})
	}
	var albums struct {
		Items []spAlbum `json:"items"`
	}
	if err := c.get(ctx, "/artists/"+id+"/albums", url.Values{"include_groups": {"album,single"}, "limit": {"50"}, "market": {"FR"}}, &albums); err != nil {
		return ArtistPage{}, err
	}
	for _, al := range albums.Items {
		page.Albums = append(page.Albums, Item{ID: al.URI, Kind: "album", Title: al.Name, Subtitle: year(al.ReleaseDate), ArtworkURL: firstImage(al.Images)})
	}
	return page, nil
}
