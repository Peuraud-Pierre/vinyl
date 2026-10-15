//go:build windows

package winmedia

import (
	"errors"
	"runtime"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/saltosystems/winrt-go/windows/foundation"
)

// Tous les appels WinRT passent par un fil système unique, initialisé une fois (RoInitialize).
var jobs = make(chan func(), 16)

func init() {
	go func() {
		runtime.LockOSThread()
		if err := ole.RoInitialize(1); err != nil { // 1 = multithread (MTA)
			panic(err)
		}
		for f := range jobs {
			f()
		}
	}()
}

// call exécute f sur le fil WinRT et attend son résultat.
func call(f func() error) error {
	done := make(chan error, 1)
	jobs <- func() { done <- f() }
	return <-done
}

// await attend la fin d'une opération asynchrone WinRT (≤ 3 s) et renvoie son résultat brut.
func await(op *foundation.IAsyncOperation) (unsafe.Pointer, error) {
	if op == nil {
		return nil, errors.New("winrt: opération nulle")
	}
	defer op.Release()
	itf, err := op.QueryInterface(ole.NewGUID(foundation.GUIDIAsyncInfo))
	if err != nil {
		return nil, err
	}
	info := (*foundation.IAsyncInfo)(unsafe.Pointer(itf))
	defer info.Release()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		st, err := info.GetStatus()
		if err != nil {
			return nil, err
		}
		switch st {
		case foundation.AsyncStatusCompleted:
			return op.GetResults()
		case foundation.AsyncStatusError, foundation.AsyncStatusCanceled:
			return nil, errors.New("winrt: opération échouée")
		}
	}
	return nil, errors.New("winrt: délai dépassé")
}
