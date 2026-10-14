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

// ITunes interroge l'API publique iTunes Search (sans clé) pour les pages artiste Apple Music.
type ITunes struct {
	BaseURL string
	HTTP    *http.Client
	Country string
}

func NewITunes() *ITunes {
	return &ITunes{BaseURL: "https://itunes.apple.com", HTTP: &http.Client{Timeout: 10 * time.Second}, Country: "FR"}
}

type itunesResult struct {
	WrapperType       string `json:"wrapperType"`
	ArtistID          int64  `json:"artistId"`
	ArtistName        string `json:"artistName"`
	ArtistLinkURL     string `json:"artistLinkUrl"`
	TrackID           int64  `json:"trackId"`
	TrackName         string `json:"trackName"`
	CollectionID      int64  `json:"collectionId"`
	CollectionName    string `json:"collectionName"`
	ReleaseDate       string `json:"releaseDate"`
	ArtworkURL100     string `json:"artworkUrl100"`
	TrackViewURL      string `json:"trackViewUrl"`
	CollectionViewURL string `json:"collectionViewUrl"`
}

// hd demande la pochette en 600×600 au lieu de 100×100.
func hd(u string) string { return strings.Replace(u, "100x100bb", "600x600bb", 1) }

func year(date string) string {
	if len(date) >= 4 {
		return date[:4]
	}
	return date
}

func (c *ITunes) get(ctx context.Context, path string, q url.Values) ([]itunesResult, error) {
	country := c.Country
	if country == "" {
		country = "FR"
	}
	q.Set("country", country)
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+path+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("itunes: HTTP %d", resp.StatusCode)
	}
	var body struct {
		Results []itunesResult `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, err
	}
	return body.Results, nil
}

func (c *ITunes) Artist(ctx context.Context, name string) (ArtistPage, error) {
	found, err := c.get(ctx, "/search", url.Values{"term": {name}, "entity": {"musicArtist"}, "limit": {"1"}})
	if err != nil {
		return ArtistPage{}, err
	}
	if len(found) == 0 {
		return ArtistPage{}, fmt.Errorf("artiste introuvable : %s", name)
	}
	a := found[0]
	id := strconv.FormatInt(a.ArtistID, 10)
	page := ArtistPage{Source: "applemusic", Name: a.ArtistName, URL: a.ArtistLinkURL}

	songs, err := c.get(ctx, "/lookup", url.Values{"id": {id}, "entity": {"song"}, "limit": {"10"}})
	if err != nil {
		return ArtistPage{}, err
	}
	for _, s := range songs {
		if s.WrapperType == "track" {
			page.TopTracks = append(page.TopTracks, Item{ID: strconv.FormatInt(s.TrackID, 10), Kind: "track",
				Title: s.TrackName, Subtitle: s.CollectionName, ArtworkURL: hd(s.ArtworkURL100), URL: s.TrackViewURL})
		}
	}
	albums, err := c.get(ctx, "/lookup", url.Values{"id": {id}, "entity": {"album"}, "limit": {"50"}})
	if err != nil {
		return ArtistPage{}, err
	}
	for _, al := range albums {
		if al.WrapperType == "collection" {
			page.Albums = append(page.Albums, Item{ID: strconv.FormatInt(al.CollectionID, 10), Kind: "album",
				Title: al.CollectionName, Subtitle: year(al.ReleaseDate), ArtworkURL: hd(al.ArtworkURL100), URL: al.CollectionViewURL})
		}
	}
	switch {
	case len(page.Albums) > 0:
		page.ImageURL = page.Albums[0].ArtworkURL
	case len(page.TopTracks) > 0:
		page.ImageURL = page.TopTracks[0].ArtworkURL
	}
	return page, nil
}
