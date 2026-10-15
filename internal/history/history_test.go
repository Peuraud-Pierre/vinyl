package history

import (
	"path/filepath"
	"strconv"
	"testing"
)

func e(id string) Entry {
	return Entry{Source: "applemusic", TrackID: id, Title: "T" + id, Artist: "A", ArtworkURL: "/artwork/" + id}
}

func ids(l []Entry) string {
	s := ""
	for _, x := range l {
		s += x.TrackID
	}
	return s
}

// Comme des disques posés après les avoir écoutés : le morceau en cours n'est pas sur la pile ;
// il y arrive, au-dessus, quand le suivant commence.
func TestFinishedTrackGoesOnTopOfThePile(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "h.json"))
	if _, changed := s.Playing(e("1")); changed {
		t.Error("le premier morceau en cours ne change pas la pile")
	}
	s.Playing(e("1")) // même morceau, sondage suivant
	l, changed := s.Playing(e("2"))
	if !changed || ids(l) != "1" {
		t.Fatalf("pile = %q (changed=%v)", ids(l), changed)
	}
	l, _ = s.Playing(e("3"))
	if ids(l) != "21" {
		t.Errorf("le plus récent au-dessus : %q", ids(l))
	}
}

func TestPileKeepsSevenAndNoDuplicates(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "h.json"))
	for i := 1; i <= 10; i++ {
		s.Playing(e(strconv.Itoa(i % 10)))
	}
	l := s.List()
	if len(l) != Max || ids(l) != "9876543" {
		t.Fatalf("pile = %q", ids(l))
	}
	s.Playing(e("5")) // 0 posé sur la pile ; 5 repris pour être écouté : retiré de la pile
	if got := ids(s.List()); got != "0987643" {
		t.Errorf("sans doublon : %q", got)
	}
}

// La pochette d'Apple Music arrive parfois après le début du morceau : on garde la plus récente.
func TestCurrentTrackDetailsAreRefreshed(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "h.json"))
	first := e("1")
	first.ArtworkURL = ""
	s.Playing(first)
	s.Playing(e("1"))
	l, _ := s.Playing(e("2"))
	if l[0].ArtworkURL != "/artwork/1" {
		t.Errorf("pochette : %q", l[0].ArtworkURL)
	}
}

func TestPileSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "h.json")
	s := NewStore(path)
	s.Playing(e("1"))
	s.Playing(e("2"))
	r := NewStore(path)
	r.Load()
	if ids(r.List()) != "1" {
		t.Fatalf("pile relue : %q", ids(r.List()))
	}
	// Le morceau en cours avant l'arrêt continue après le relancement : il n'est pas posé.
	if _, changed := r.Playing(e("2")); changed {
		t.Error("même morceau qu'avant le relancement : pile inchangée")
	}
}

func TestEmptyTrackIDIgnored(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "h.json"))
	s.Playing(e("1"))
	if _, changed := s.Playing(Entry{Source: "applemusic"}); changed || len(s.List()) != 0 {
		t.Error("flux sans identifiant : ignoré")
	}
}

// Reprendre un ancien disque : ceux posés au-dessus sont retirés, il devient le morceau en cours
// (le morceau qui jouait n'est pas reposé sur la pile).
func TestTakeFromRemovesTheOnesAbove(t *testing.T) {
	path := filepath.Join(t.TempDir(), "h.json")
	s := NewStore(path)
	for _, id := range []string{"1", "2", "3", "4", "5"} {
		s.Playing(e(id))
	} // pile 4321, en cours 5
	if _, steps := s.TakeFrom("applemusic", "2"); steps != 3 {
		t.Errorf("2 est 3 morceaux en arrière (4, 3, puis 2) : %d", steps)
	}
	if got := ids(s.List()); got != "1" {
		t.Fatalf("pile = %q, attendu les disques sous 2", got)
	}
	if _, changed := s.Playing(e("2")); changed {
		t.Error("2 est le morceau en cours : quand il démarre, rien n'est reposé")
	}
	r := NewStore(path)
	r.Load()
	if ids(r.List()) != "1" {
		t.Errorf("enregistré : %q", ids(r.List()))
	}
}

func TestTakeFromUndo(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "h.json"))
	for _, id := range []string{"1", "2", "3"} {
		s.Playing(e(id))
	}
	undo, _ := s.TakeFrom("applemusic", "1")
	undo() // lecture impossible : pile remise comme avant
	if got := ids(s.List()); got != "21" {
		t.Errorf("pile = %q", got)
	}
	if _, changed := s.Playing(e("3")); changed {
		t.Error("le morceau en cours est aussi rétabli")
	}
	undoAbsent, steps := s.TakeFrom("applemusic", "absent") // inconnu : rien ne change
	undoAbsent()
	if steps != 0 {
		t.Errorf("absent : %d pas", steps)
	}
	if got := ids(s.List()); got != "21" {
		t.Errorf("pile = %q", got)
	}
}
