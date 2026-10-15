//go:build windows

package winmedia

import (
	"time"

	"github.com/saltosystems/winrt-go/windows/foundation"
	"github.com/saltosystems/winrt-go/windows/media"
	"github.com/saltosystems/winrt-go/windows/media/control"

	"vinyl/internal/player"
)

type systemSessions struct {
	mgr *control.GlobalSystemMediaTransportControlsSessionManager
}

// NewSystemSessions ouvre le gestionnaire des contrôles multimédia de Windows.
func NewSystemSessions() (Sessions, error) {
	var s systemSessions
	err := call(func() error {
		op, err := control.GlobalSystemMediaTransportControlsSessionManagerRequestAsync()
		if err != nil {
			return err
		}
		res, err := await(op)
		if err != nil {
			return err
		}
		s.mgr = (*control.GlobalSystemMediaTransportControlsSessionManager)(res)
		return nil
	})
	return &s, err
}

func (s *systemSessions) List() ([]Session, error) {
	var out []Session
	err := call(func() error {
		vec, err := s.mgr.GetSessions()
		if err != nil {
			return err
		}
		defer vec.Release()
		n, err := vec.GetSize()
		if err != nil {
			return err
		}
		for i := uint32(0); i < n; i++ {
			p, err := vec.GetAt(i)
			if err != nil {
				return err
			}
			ses := (*control.GlobalSystemMediaTransportControlsSession)(p)
			id, err := ses.GetSourceAppUserModelId()
			if err != nil {
				continue
			}
			out = append(out, &systemSession{s: ses, id: id})
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

func (x *systemSession) Info() (Info, error) {
	var in Info
	err := call(func() error {
		op, err := x.s.TryGetMediaPropertiesAsync()
		if err != nil {
			return err
		}
		res, err := await(op)
		if err != nil {
			return err
		}
		props := (*control.GlobalSystemMediaTransportControlsSessionMediaProperties)(res)
		in.Title, _ = props.GetTitle()
		in.Artist, _ = props.GetArtist()
		in.Album, _ = props.GetAlbumTitle()
		if tl, err := x.s.GetTimelineProperties(); err == nil {
			pos, _ := tl.GetPosition()
			end, _ := tl.GetEndTime()
			start, _ := tl.GetStartTime()
			upd, _ := tl.GetLastUpdatedTime()
			in.Position, in.Duration = ticks(pos.Duration), ticks(end.Duration-start.Duration)
			// DateTime : unités de 100 ns depuis le 1er janvier 1601 (UTC).
			in.UpdatedAt = time.Unix(0, (upd.UniversalTime-116444736000000000)*100)
		}
		if pi, err := x.s.GetPlaybackInfo(); err == nil {
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
				in.Shuffle = uintptr(v)&0xff != 0 // IReference<bool>
			}
			if ref, err := pi.GetAutoRepeatMode(); err == nil && ref != nil {
				v, _ := ref.GetValue()
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
		_, err = await(o)
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

// Thumbnail : Task 6.
func (x *systemSession) Thumbnail() ([]byte, error) { return nil, nil }
