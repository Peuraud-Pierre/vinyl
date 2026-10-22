//go:build windows

package sysvolume

import (
	"context"
	"fmt"
	"math"
	"runtime"
	"syscall"
	"unsafe"

	ole "github.com/go-ole/go-ole"
)

// Volume principal du périphérique de sortie par défaut (Core Audio : IMMDeviceEnumerator →
// IMMDevice → IAudioEndpointVolume), appelé par sa table de méthodes COM.
var (
	clsidMMDeviceEnumerator = ole.NewGUID("{BCDE0395-E52F-467C-8E3D-C4579291692E}")
	iidIMMDeviceEnumerator  = ole.NewGUID("{A95664D2-9614-4F35-A746-DE8DB63617E6}")
	iidIAudioEndpointVolume = ole.NewGUID("{5CDF2C82-841E-4546-9722-0CF74078229A}")
)

// Rangs dans les tables de méthodes (après QueryInterface, AddRef, Release).
const (
	vtRelease                    = 2
	vtGetDefaultAudioEndpoint    = 4 // IMMDeviceEnumerator
	vtActivate                   = 3 // IMMDevice
	vtSetMasterVolumeLevelScalar = 7 // IAudioEndpointVolume
	vtGetMasterVolumeLevelScalar = 9
	vtSetMute                    = 14
	vtGetMute                    = 15
)

func method(obj unsafe.Pointer, i int) uintptr {
	vtbl := *(*unsafe.Pointer)(obj)
	return *(*uintptr)(unsafe.Add(vtbl, uintptr(i)*unsafe.Sizeof(uintptr(0))))
}

func call(obj unsafe.Pointer, i int, args ...uintptr) error {
	hr, _, _ := syscall.SyscallN(method(obj, i), append([]uintptr{uintptr(obj)}, args...)...)
	if int32(hr) < 0 {
		return ole.NewError(hr)
	}
	return nil
}

func release(obj unsafe.Pointer) { syscall.SyscallN(method(obj, vtRelease), uintptr(obj)) }

// withEndpoint ouvre le volume de la sortie par défaut sur un fil système initialisé pour COM.
func withEndpoint(f func(ep unsafe.Pointer) error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("sysvolume: %v", r)
		}
	}()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if e := ole.CoInitializeEx(0, ole.COINIT_MULTITHREADED); e == nil {
		defer ole.CoUninitialize()
	}
	unk, err := ole.CreateInstance(clsidMMDeviceEnumerator, iidIMMDeviceEnumerator)
	if err != nil {
		return err
	}
	enum := unsafe.Pointer(unk)
	defer release(enum)
	var dev unsafe.Pointer
	if err := call(enum, vtGetDefaultAudioEndpoint, 0 /* eRender */, 0 /* eConsole */, uintptr(unsafe.Pointer(&dev))); err != nil {
		return err
	}
	defer release(dev)
	var ep unsafe.Pointer
	if err := call(dev, vtActivate, uintptr(unsafe.Pointer(iidIAudioEndpointVolume)), uintptr(ole.CLSCTX_ALL), 0, uintptr(unsafe.Pointer(&ep))); err != nil {
		return err
	}
	defer release(ep)
	return f(ep)
}

func Get(context.Context) (int, error) {
	var v int
	err := withEndpoint(func(ep unsafe.Pointer) error {
		var muted int32
		if err := call(ep, vtGetMute, uintptr(unsafe.Pointer(&muted))); err == nil && muted != 0 {
			return nil
		}
		var level float32
		if err := call(ep, vtGetMasterVolumeLevelScalar, uintptr(unsafe.Pointer(&level))); err != nil {
			return err
		}
		v = clamp(int(math.Round(float64(level) * 100)))
		return nil
	})
	return v, err
}

// Set règle le volume et rétablit le son. Le réel passe en registre entier : l'appel système le recopie
// aussi dans les registres XMM (convention Windows x64).
func Set(_ context.Context, v int) error {
	return withEndpoint(func(ep unsafe.Pointer) error {
		if err := call(ep, vtSetMasterVolumeLevelScalar, uintptr(math.Float32bits(float32(clamp(v))/100)), 0); err != nil {
			return err
		}
		return call(ep, vtSetMute, 0, 0)
	})
}
