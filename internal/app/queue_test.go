package app

import (
	"context"
	"testing"

	"vinyl/internal/catalog"
	"vinyl/internal/player"
)

// fakeCatalog : un album de trois titres A, B, C.
type fakeCatalog struct{}

func (fakeCatalog) Library(context.Context, string, int) (catalog.Page, error) {
	return catalog.Page{}, nil
}
func (fakeCatalog) Artist(context.Context, string, string) (catalog.ArtistPage, error) {
	return catalog.ArtistPage{}, nil
}
func (fakeCatalog) Tracks(context.Context, string) (catalog.Collection, error) {
	return catalog.Collection{Tracks: []player.Track{{ID: "A", Title: "a"}, {ID: "B", Title: "b"}, {ID: "C", Title: "c"}}}, nil
}

func albumService(p *fakeResumer) *VinylService {
	s := serviceWith(p)
	s.SetCatalog("applemusic", fakeCatalog{})
	return s
}

func stopped() player.Snapshot {
	return player.Snapshot{NowPlaying: player.NowPlaying{Source: "applemusic", State: player.StateStopped}}
}

// Fin naturelle du morceau id (Musique s'arrête : il était joué « une fois »).
func finish(s *VinylService, id string) {
	s.ObservePlayback(playing(id, 1000, 200000))
	s.ObservePlayback(playing(id, 199000, 200000))
	s.ObservePlayback(stopped())
}

// Sans créer de playlist dans Musique : Vinyl joue l'album titre par titre, dans l'ordre.
func TestAlbumPlaysTrackByTrackWithoutPlaylist(t *testing.T) {
	p := &fakeResumer{}
	s := albumService(p)
	if err := s.PlayFrom("applemusic", "album:X", "B", false); err != nil {
		t.Fatal(err)
	}
	if q, _ := s.Queue(); len(q.Upcoming) != 1 || q.Upcoming[0].ID != "C" {
		t.Errorf("file = %+v", q.Upcoming)
	}
	finish(s, "B")
	finish(s, "C")
	if got := p.alone; len(got) != 2 || got[0] != "B" || got[1] != "C" {
		t.Errorf("joués : %v", got)
	}
	s.ObservePlayback(stopped())
	if len(p.alone) != 2 {
		t.Error("fin de l'album : plus rien")
	}
}

func TestAlbumRepeatAllLoops(t *testing.T) {
	p := &fakeResumer{}
	p.repeat = player.RepeatAll
	s := albumService(p)
	s.PlayFrom("applemusic", "album:X", "C", false)
	finish(s, "C")
	if got := p.alone; len(got) != 2 || got[1] != "A" {
		t.Errorf("joués : %v", got)
	}
}

func TestNextAndPreviousFollowTheAlbum(t *testing.T) {
	p := &fakeResumer{}
	s := albumService(p)
	s.PlayFrom("applemusic", "album:X", "B", false)
	s.ObservePlayback(playing("B", 1000, 200000))
	if err := s.Next(); err != nil {
		t.Fatal(err)
	}
	s.ObservePlayback(playing("C", 500, 200000))
	if err := s.Previous(); err != nil {
		t.Fatal(err)
	}
	if got := p.alone; len(got) != 3 || got[1] != "C" || got[2] != "B" {
		t.Errorf("joués : %v", got)
	}
}

func TestShuffledAlbumPlaysEveryTrackOnce(t *testing.T) {
	p := &fakeResumer{}
	s := albumService(p)
	s.PlayFrom("applemusic", "album:X", "", true)
	for range 3 {
		finish(s, p.alone[len(p.alone)-1])
	}
	seen := map[string]bool{}
	for _, id := range p.alone {
		seen[id] = true
	}
	if len(p.alone) != 3 || len(seen) != 3 {
		t.Errorf("joués : %v", p.alone)
	}
}

func TestPlayItemAlbumUsesTheQueue(t *testing.T) {
	p := &fakeResumer{}
	s := albumService(p)
	if err := s.PlayItem("applemusic", "album:X"); err != nil {
		t.Fatal(err)
	}
	if len(p.alone) != 1 || p.alone[0] != "A" {
		t.Errorf("joués : %v", p.alone)
	}
}

// L'utilisateur choisit autre chose dans Musique : Vinyl arrête de piloter la suite.
func TestAlbumQueueStopsWhenTheUserPicksSomethingElse(t *testing.T) {
	p := &fakeResumer{}
	s := albumService(p)
	s.PlayFrom("applemusic", "album:X", "A", false)
	s.ObservePlayback(playing("A", 30000, 200000))
	s.ObservePlayback(playing("ZZ", 500, 200000))
	s.ObservePlayback(stopped())
	if len(p.alone) != 1 {
		t.Errorf("joués : %v", p.alone)
	}
}
