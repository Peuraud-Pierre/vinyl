package winmedia

import (
	"context"
	"time"

	"vinyl/internal/player"
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

// waitChange attend (≤ 1,5 s) que le morceau en cours ne soit plus prev : après « suivant », le
// changement peut être lent (fondus), et appuyer de nouveau trop tôt ferait dépasser le morceau visé.
func (p *Player) waitChange(prev string) {
	for range 15 {
		if p.current() != prev {
			return
		}
		p.sleep(100 * time.Millisecond)
	}
}

// nextUntil appuie sur « suivant » jusqu'à ce que trackID soit le morceau en cours (au plus max fois) :
// on compte les vrais changements de morceau, pas les appuis. Renvoie true s'il est atteint.
func (p *Player) nextUntil(s Session, trackID string, max int) bool {
	for range max {
		cur := p.current()
		if cur == trackID {
			return true
		}
		if err := s.Next(); err != nil {
			return false
		}
		p.waitChange(cur)
	}
	return p.waitFor(trackID)
}

// ReplayBack réécoute un morceau sans toucher à la file du lecteur : « précédent » steps fois, puis un
// ou deux crans de plus si la pile est décalée ou si le premier appui n'a fait que relancer le morceau
// en cours (au-delà de 3 s). Introuvable : retour par « suivant » au morceau de départ, à sa position ;
// renvoie 0.
func (p *Player) ReplayBack(_ context.Context, trackID string, steps, posSec int) (int, error) {
	s, err := p.session()
	if err != nil {
		return 0, err
	}
	start := p.current()
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
	if p.nextUntil(s, start, taken+1) {
		s.Seek(time.Duration(posSec) * time.Second)
	}
	return 0, nil
}

// JumpForward revient au morceau interrompu par « suivant » (au plus steps + 1 appuis, en comptant
// les vrais changements de morceau) et le replace où il en était.
func (p *Player) JumpForward(_ context.Context, steps int, trackID string, posSec int) error {
	s, err := p.session()
	if err != nil {
		return err
	}
	if p.nextUntil(s, trackID, steps+1) {
		return s.Seek(time.Duration(posSec) * time.Second)
	}
	return nil
}

// SaveContext : où en est la lecture (morceau, position), pour y revenir après une réécoute.
func (p *Player) SaveContext(ctx context.Context) (player.PlaybackContext, error) {
	np, err := p.NowPlaying(ctx)
	if err != nil {
		return player.PlaybackContext{}, err
	}
	return player.PlaybackContext{TrackID: np.TrackID, PositionSec: int(np.PositionMs / 1000)}, nil
}
