package winmedia

import (
	"context"
	"time"
)

// current : identifiant du morceau en cours ("" si aucun).
func (p *Player) current() string {
	s, err := p.session()
	if err != nil {
		return ""
	}
	in, err := s.Info()
	if err != nil || in.Title == "" {
		return ""
	}
	return TrackID(in.Title, in.Artist, in.Album)
}

// waitFor attend (≤ 1,5 s) que le morceau visé soit celui en cours (fondus, changements lents).
func (p *Player) waitFor(trackID string) bool {
	for range 15 {
		if p.current() == trackID {
			return true
		}
		p.sleep(100 * time.Millisecond)
	}
	return p.current() == trackID
}

// ReplayBack réécoute un morceau sans toucher à la file du lecteur : « précédent » steps fois, puis un
// ou deux crans de plus si la pile est décalée. Introuvable : retour par « suivant » au morceau de
// départ, à sa position ; renvoie 0.
func (p *Player) ReplayBack(_ context.Context, trackID string, steps, posSec int) (int, error) {
	s, err := p.session()
	if err != nil {
		return 0, err
	}
	taken := 0
	for range steps {
		if err := s.Previous(); err != nil {
			return 0, err
		}
		taken++
		p.sleep(100 * time.Millisecond)
	}
	for extra := 0; extra <= 2; extra++ {
		if extra > 0 {
			if err := s.Previous(); err != nil {
				return 0, err
			}
			taken++
		}
		if p.waitFor(trackID) {
			return taken, nil
		}
	}
	for range taken {
		s.Next()
		p.sleep(100 * time.Millisecond)
	}
	p.sleep(300 * time.Millisecond)
	s.Seek(time.Duration(posSec) * time.Second)
	return 0, nil
}

// JumpForward revient au morceau interrompu par « suivant » et le replace où il en était.
func (p *Player) JumpForward(_ context.Context, steps int, trackID string, posSec int) error {
	s, err := p.session()
	if err != nil {
		return err
	}
	for range steps {
		if err := s.Next(); err != nil {
			return err
		}
		p.sleep(100 * time.Millisecond)
	}
	if p.waitFor(trackID) {
		return s.Seek(time.Duration(posSec) * time.Second)
	}
	return nil
}
