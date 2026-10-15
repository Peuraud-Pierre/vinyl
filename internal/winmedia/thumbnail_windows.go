//go:build windows

package winmedia

import (
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/saltosystems/winrt-go/windows/foundation"

	"vinyl/internal/winmedia/winrtgen/windows/storage/streams"
)

// readThumbnail lit la miniature d'une session : OpenReadAsync, puis DataReader sur le flux.
func (x *systemSession) readThumbnail() ([]byte, error) {
	var out []byte
	err := call(func() error {
		props, err := x.properties()
		if err != nil || props == nil {
			return err
		}
		defer props.Release()
		ref, err := props.GetThumbnail()
		if err != nil || ref == nil {
			return err // pas de pochette : rien à lire
		}
		defer ref.Release()
		openOp, err := ref.OpenReadAsync()
		if err != nil {
			return err
		}
		stream, err := await(openOp)
		if err != nil || stream == nil {
			return err
		}
		unk := (*ole.IUnknown)(stream)
		defer unk.Release()
		in, err := unk.QueryInterface(ole.NewGUID(streams.GUIDIInputStream))
		if err != nil {
			return err
		}
		defer in.Release()
		r, err := streams.DataReaderCreateDataReader((*streams.IInputStream)(unsafe.Pointer(in)))
		if err != nil || r == nil {
			return err
		}
		defer r.Release()
		const max = 4 << 20 // 4 Mo : largement assez pour une miniature
		load, err := r.LoadAsync(max)
		if err != nil {
			return err
		}
		// L'opération de chargement (IAsyncOperation<uint32>) a la même forme qu'une opération ordinaire.
		n, err := await((*foundation.IAsyncOperation)(unsafe.Pointer(load)))
		if err != nil || uint32(uintptr(n)) == 0 {
			return err // flux vide : pas de pochette
		}
		out, err = r.ReadBytes(uint32(uintptr(n)))
		return err
	})
	return out, err
}
