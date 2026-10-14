// Package catalog fournit la bibliothèque et les pages artiste pour chaque source.
package catalog

import (
	"context"
	"errors"
)

const PageSize = 50

var ErrUnsupported = errors.New("non disponible pour cette source")

type Item struct {
	ID         string `json:"id"`
	Kind       string `json:"kind"`
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle"`
	ArtworkURL string `json:"artworkUrl"`
	URL        string `json:"url"`
}

type Page struct {
	Items []Item `json:"items"`
	Next  int    `json:"next"`
}

type ArtistPage struct {
	Source    string `json:"source"`
	Name      string `json:"name"`
	ImageURL  string `json:"imageUrl"`
	URL       string `json:"url"`
	TopTracks []Item `json:"topTracks"`
	Albums    []Item `json:"albums"`
}

type Catalog interface {
	Library(ctx context.Context, kind string, offset int) (Page, error)
	Artist(ctx context.Context, name, trackID string) (ArtistPage, error)
}

func paginate(items []Item, offset int) Page {
	if offset < 0 || offset > len(items) {
		offset = len(items)
	}
	end := min(offset+PageSize, len(items))
	next := end
	if end >= len(items) {
		next = -1
	}
	return Page{Items: items[offset:end], Next: next}
}
