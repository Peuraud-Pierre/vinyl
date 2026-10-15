//go:build windows

package winmedia

import (
	"errors"
	"time"

	"github.com/saltosystems/winrt-go/windows/foundation"
	"github.com/saltosystems/winrt-go/windows/media"
	"github.com/saltosystems/winrt-go/windows/media/control"

	"vinyl/internal/player"
)

type systemSessions struct {
	mgr   *control.GlobalSystemMediaTransportControlsSessionManager
	cache map[string]*systemSession // une session par app, gardée d'un sondage à l'autre
}

// NewSystemSessions ouvre le gestionnaire des contrôles multimédia de Windows.
func NewSystemSessions() (Sessions, error) {
	s := systemSessions{cache: map[string]*systemSession{}}
	err := call(func() error {
		op, err := control.GlobalSystemMediaTransportControlsSessionManagerRequestAsync()
		if err != nil {
			return err
		}
		res, err := await(op)
		if err != nil {
			return err
		}
		if res == nil {
			return errors.New("winmedia: gestionnaire indisponible")
		}
		s.mgr = (*control.GlobalSystemMediaTransportControlsSessionManager)(res)
		return nil
	})
	return &s, err
}

// List renvoie les sessions ouvertes. Chaque objet obtenu de Windows est libéré, sauf la session
// gardée en cache (une par app) ; celles des apps fermées sont libérées.
func (s *systemSessions) List() ([]Session, error) {
	var out []Session
	err := call(func() error {
		vec, err := s.mgr.GetSessions()
		if err != nil {
			return err
		}
		if vec == nil {
			return nil
		}
		defer vec.Release()
		n, err := vec.GetSize()
		if err != nil {
			return err
		}
		seen := map[string]bool{}
		for i := uint32(0); i < n; i++ {
			p, err := vec.GetAt(i)
			if err != nil || p == nil {
				continue
			}
			ses := (*control.GlobalSystemMediaTransportControlsSession)(p)
			id, err := ses.GetSourceAppUserModelId()
			if err != nil || seen[id] {
				ses.Release()
				continue
			}
			seen[id] = true
			if old, ok := s.cache[id]; ok {
				ses.Release() // déjà en cache
				out = append(out, old)
				continue
			}
			x := &systemSession{s: ses, id: id}
			s.cache[id] = x
			out = append(out, x)
		}
		for id, x := range s.cache {
			if !seen[id] { // app fermée
				x.s.Release()
				delete(s.cache, id)
			}
		}
		return nil
	})
	return out, err
}

type systemSession struct {
	s  *control.GlobalSystemMediaTransportControlsSession
	id string
}

func (x *systemSession) AppID() string { return x.id }

// ticks : unités de 100 ns de WinRT.
func ticks(v int64) time.Duration { return time.Duration(v) * 100 }

// fileTimeEpoch : 1er janvier 1970 en unités de 100 ns depuis 1601 (DateTime de WinRT).
const fileTimeEpoch = 116444736000000000

// properties : propriétés du morceau en cours (à libérer par l'appelant) ; nil si aucune.
func (x *systemSession) properties() (*control.GlobalSystemMediaTransportControlsSessionMediaProperties, error) {
	op, err := x.s.TryGetMediaPropertiesAsync()
	if err != nil {
		return nil, err
	}
	res, err := await(op)
	if err != nil || res == nil {
		return nil, err
	}
	return (*control.GlobalSystemMediaTransportControlsSessionMediaProperties)(res), nil
}

func (x *systemSession) Info() (Info, error) {
	var in Info
	err := call(func() error {
		props, err := x.properties()
		if err != nil {
			return err
		}
		if props != nil {
			in.Title, _ = props.GetTitle()
			in.Artist, _ = props.GetArtist()
			in.Album, _ = props.GetAlbumTitle()
			props.Release()
		}
		if tl, err := x.s.GetTimelineProperties(); err == nil && tl != nil {
			pos, _ := tl.GetPosition()
			end, _ := tl.GetEndTime()
			start, _ := tl.GetStartTime()
			upd, _ := tl.GetLastUpdatedTime()
			tl.Release()
			in.Position, in.Duration = ticks(pos.Duration-start.Duration), ticks(end.Duration-start.Duration)
			if upd.UniversalTime > fileTimeEpoch { // 0 : l'app ne publie pas de relevé
				in.UpdatedAt = time.Unix(0, (upd.UniversalTime-fileTimeEpoch)*100)
			}
		}
		if pi, err := x.s.GetPlaybackInfo(); err == nil && pi != nil {
			defer pi.Release()
			st, _ := pi.GetPlaybackStatus()
			switch st {
			case control.GlobalSystemMediaTransportControlsSessionPlaybackStatusPlaying:
				in.Status = player.StatePlaying
			case control.GlobalSystemMediaTransportControlsSessionPlaybackStatusPaused:
				in.Status = player.StatePaused
			default:
				in.Status = player.StateStopped
			}
			if ref, err := pi.GetIsShuffleActive(); err == nil && ref != nil {
				v, _ := ref.GetValue()
				ref.Release()
				in.Shuffle = uintptr(v)&0xff != 0 // IReference<bool>
			}
			if ref, err := pi.GetAutoRepeatMode(); err == nil && ref != nil {
				v, _ := ref.GetValue()
				ref.Release()
				switch media.MediaPlaybackAutoRepeatMode(int32(uintptr(v))) { // IReference<enum>
				case media.MediaPlaybackAutoRepeatModeTrack:
					in.Repeat = player.RepeatOne
				case media.MediaPlaybackAutoRepeatModeList:
					in.Repeat = player.RepeatAll
				default:
					in.Repeat = player.RepeatOff
				}
			}
		}
		return nil
	})
	return in, err
}

// try exécute une commande asynchrone de la session et attend sa fin.
func (x *systemSession) try(op func() (*foundation.IAsyncOperation, error)) error {
	return call(func() error {
		o, err := op()
		if err != nil {
			return err
		}
		res, err := await(o)
		if err == nil && uintptr(res)&0xff == 0 { // IAsyncOperation<bool> : false = refusée par l'app
			err = errors.New("commande refusée par l'app")
		}
		return err
	})
}

func (x *systemSession) TogglePlayPause() error { return x.try(x.s.TryTogglePlayPauseAsync) }
func (x *systemSession) Next() error            { return x.try(x.s.TrySkipNextAsync) }
func (x *systemSession) Previous() error        { return x.try(x.s.TrySkipPreviousAsync) }
func (x *systemSession) Seek(pos time.Duration) error {
	return x.try(func() (*foundation.IAsyncOperation, error) {
		return x.s.TryChangePlaybackPositionAsync(int64(pos / 100))
	})
}
func (x *systemSession) SetShuffle(on bool) error {
	return x.try(func() (*foundation.IAsyncOperation, error) { return x.s.TryChangeShuffleActiveAsync(on) })
}
func (x *systemSession) SetRepeat(m player.RepeatMode) error {
	mode := media.MediaPlaybackAutoRepeatModeNone
	switch m {
	case player.RepeatOne:
		mode = media.MediaPlaybackAutoRepeatModeTrack
	case player.RepeatAll:
		mode = media.MediaPlaybackAutoRepeatModeList
	}
	return x.try(func() (*foundation.IAsyncOperation, error) { return x.s.TryChangeAutoRepeatModeAsync(mode) })
}

// Thumbnail : miniature de la pochette (thumbnail_windows.go).
func (x *systemSession) Thumbnail() ([]byte, error) { return x.readThumbnail() }
