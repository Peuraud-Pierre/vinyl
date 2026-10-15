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
var (
	jobs    = make(chan func(), 16)
	initErr error // initialisation impossible : chaque appel renvoie cette erreur (pas de plantage)
)

func init() {
	ready := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		// 1 = multithread (MTA). go-ole signale aussi S_FALSE (« déjà initialisé ») comme une erreur.
		if err := ole.RoInitialize(1); err != nil && !isSFalse(err) {
			initErr = err
		}
		close(ready)
		for f := range jobs {
			f()
		}
	}()
	<-ready
}

// isSFalse : S_FALSE (1) n'est pas un échec.
func isSFalse(err error) bool {
	var oe *ole.OleError
	return errors.As(err, &oe) && oe.Code() == 1
}

// call exécute f sur le fil WinRT et attend son résultat ; une panique devient une erreur.
func call(f func() error) error {
	if initErr != nil {
		return initErr
	}
	done := make(chan error, 1)
	jobs <- func() { done <- safeRun(f) }
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
	defer func() {
		if st, _ := info.GetStatus(); st == foundation.AsyncStatusStarted {
			info.Cancel() // délai dépassé : on n'attend plus ce résultat
		}
	}()
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
