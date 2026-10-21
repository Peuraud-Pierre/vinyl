//go:build darwin && cgo

// Package audiotap capte le son de sortie du Mac (Core Audio, « process tap », macOS 14.2 et plus) et en
// tire l'énergie de quelques bandes de fréquences, pour l'égaliseur de la platine. Rien n'est enregistré :
// chaque bloc de son est analysé puis oublié. La première capture demande l'autorisation d'enregistrer
// l'audio ; refusée, le son capté reste muet.
package audiotap

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation -framework CoreAudio -framework Accelerate
#import <Foundation/Foundation.h>
#import <CoreAudio/CoreAudio.h>
#import <CoreAudio/CATapDescription.h>
#import <CoreAudio/AudioHardwareTapping.h>
#import <Accelerate/Accelerate.h>
#import <os/lock.h>
#include <stdlib.h>
#include <string.h>

#define VY_LOG2N 10
#define VY_N (1 << VY_LOG2N)
#define VY_BANDS 12

static AudioObjectID vyTap = kAudioObjectUnknown, vyAgg = kAudioObjectUnknown;
static AudioDeviceIOProcID vyProc = NULL;
static os_unfair_lock vyLock = OS_UNFAIR_LOCK_INIT;
static float vyBands[VY_BANDS];
static unsigned long long vySeq;
static float vyBuf[VY_N], vyWin[VY_N], vyRe[VY_N / 2], vyIm[VY_N / 2], vyPow[VY_N / 2];
static int vyFill;
static FFTSetup vyFFT;
static double vyRate = 48000;
static int vyChannels = 2, vyInterleaved = 1;

// Spectre d'un bloc plein : puissance moyenne (dB) de bandes réparties en octaves de 50 Hz à 14 kHz.
static void vyAnalyse(void) {
	float w[VY_N];
	vDSP_vmul(vyBuf, 1, vyWin, 1, w, 1, VY_N);
	DSPSplitComplex z = { vyRe, vyIm };
	vDSP_ctoz((const DSPComplex *)w, 2, &z, 1, VY_N / 2);
	vDSP_fft_zrip(vyFFT, &z, 1, VY_LOG2N, FFT_FORWARD);
	vyIm[0] = 0; // case 0 : composante continue et Nyquist mêlées, inutiles ici
	vDSP_zvmags(&z, 1, vyPow, 1, VY_N / 2);
	float out[VY_BANDS];
	for (int b = 0; b < VY_BANDS; b++) {
		double lo = 50 * pow(280.0, (double)b / VY_BANDS), hi = 50 * pow(280.0, (double)(b + 1) / VY_BANDS);
		int i0 = (int)(lo * VY_N / vyRate), i1 = (int)(hi * VY_N / vyRate);
		if (i0 < 1) i0 = 1;
		if (i1 <= i0) i1 = i0 + 1;
		if (i1 > VY_N / 2) i1 = VY_N / 2;
		double p = 0;
		for (int i = i0; i < i1; i++) p += vyPow[i];
		p /= (i1 - i0) * (double)VY_N * VY_N;
		out[b] = (float)(10 * log10(p + 1e-12));
	}
	os_unfair_lock_lock(&vyLock);
	memcpy(vyBands, out, sizeof out);
	vySeq++;
	os_unfair_lock_unlock(&vyLock);
}

// Son reçu (flottants 32 bits) : mélangé en mono, puis analysé par blocs de VY_N échantillons.
static void vyFeed(const AudioBufferList *in) {
	if (!in || in->mNumberBuffers == 0) return;
	UInt32 frames = vyInterleaved ? in->mBuffers[0].mDataByteSize / (sizeof(float) * vyChannels)
	                              : in->mBuffers[0].mDataByteSize / sizeof(float);
	for (UInt32 f = 0; f < frames; f++) {
		float v = 0;
		if (vyInterleaved) {
			const float *d = in->mBuffers[0].mData;
			for (int c = 0; c < vyChannels; c++) v += d[f * vyChannels + c];
			v /= vyChannels;
		} else {
			for (UInt32 c = 0; c < in->mNumberBuffers; c++) v += ((const float *)in->mBuffers[c].mData)[f];
			v /= in->mNumberBuffers;
		}
		vyBuf[vyFill++] = v;
		if (vyFill == VY_N) { vyAnalyse(); vyFill = 0; }
	}
}

static char *vyErr(const char *what, OSStatus st) {
	char *s = NULL;
	asprintf(&s, "%s (erreur %d)", what, (int)st);
	return s;
}

static void vyStop(void) {
	if (vyAgg != kAudioObjectUnknown) {
		if (vyProc) { AudioDeviceStop(vyAgg, vyProc); AudioDeviceDestroyIOProcID(vyAgg, vyProc); }
		AudioHardwareDestroyAggregateDevice(vyAgg);
	}
	if (@available(macOS 14.2, *)) {
		if (vyTap != kAudioObjectUnknown) AudioHardwareDestroyProcessTap(vyTap);
	}
	vyProc = NULL;
	vyAgg = vyTap = kAudioObjectUnknown;
}

static int vyAvailable(void) {
	if (@available(macOS 14.2, *)) return 1;
	return 0;
}

// vyStart renvoie NULL si la capture tourne, sinon un message d'erreur à libérer.
static char *vyStart(void) {
	if (vyAgg != kAudioObjectUnknown) return NULL;
	if (@available(macOS 14.2, *)) {
		if (!vyFFT) {
			vyFFT = vDSP_create_fftsetup(VY_LOG2N, kFFTRadix2);
			vDSP_hann_window(vyWin, VY_N, vDSP_HANN_NORM);
		}
		CATapDescription *d = [[CATapDescription alloc] initStereoGlobalTapButExcludeProcesses:@[]];
		d.name = @"Vinyl égaliseur";
		d.privateTap = YES;
		d.muteBehavior = CATapUnmuted;
		OSStatus st = AudioHardwareCreateProcessTap(d, &vyTap);
		if (st != noErr) { vyTap = kAudioObjectUnknown; return vyErr("capture du son impossible", st); }

		AudioObjectPropertyAddress a = { kAudioTapPropertyFormat, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain };
		AudioStreamBasicDescription f = {0};
		UInt32 sz = sizeof f;
		st = AudioObjectGetPropertyData(vyTap, &a, 0, NULL, &sz, &f);
		if (st != noErr || f.mFormatID != kAudioFormatLinearPCM || !(f.mFormatFlags & kAudioFormatFlagIsFloat) || f.mBitsPerChannel != 32) {
			vyStop();
			return vyErr("format du son capté inattendu", st);
		}
		vyRate = f.mSampleRate > 0 ? f.mSampleRate : 48000;
		vyChannels = f.mChannelsPerFrame > 0 ? (int)f.mChannelsPerFrame : 2;
		vyInterleaved = !(f.mFormatFlags & kAudioFormatFlagIsNonInterleaved);

		AudioObjectID out = kAudioObjectUnknown;
		a = (AudioObjectPropertyAddress){ kAudioHardwarePropertyDefaultSystemOutputDevice, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain };
		sz = sizeof out;
		st = AudioObjectGetPropertyData(kAudioObjectSystemObject, &a, 0, NULL, &sz, &out);
		if (st != noErr) { vyStop(); return vyErr("sortie audio introuvable", st); }
		CFStringRef uid = NULL;
		a = (AudioObjectPropertyAddress){ kAudioDevicePropertyDeviceUID, kAudioObjectPropertyScopeGlobal, kAudioObjectPropertyElementMain };
		sz = sizeof uid;
		st = AudioObjectGetPropertyData(out, &a, 0, NULL, &sz, &uid);
		if (st != noErr || !uid) { vyStop(); return vyErr("sortie audio introuvable", st); }
		NSString *outUID = (__bridge_transfer NSString *)uid;

		// Périphérique agrégé privé : la sortie actuelle et la capture, pour recevoir le son.
		NSDictionary *agg = @{
			@kAudioAggregateDeviceNameKey: @"Vinyl égaliseur",
			@kAudioAggregateDeviceUIDKey: [NSUUID UUID].UUIDString,
			@kAudioAggregateDeviceMainSubDeviceKey: outUID,
			@kAudioAggregateDeviceIsPrivateKey: @YES,
			@kAudioAggregateDeviceIsStackedKey: @NO,
			@kAudioAggregateDeviceTapAutoStartKey: @YES,
			@kAudioAggregateDeviceSubDeviceListKey: @[ @{ @kAudioSubDeviceUIDKey: outUID } ],
			@kAudioAggregateDeviceTapListKey: @[ @{ @kAudioSubTapDriftCompensationKey: @YES, @kAudioSubTapUIDKey: d.UUID.UUIDString } ],
		};
		st = AudioHardwareCreateAggregateDevice((__bridge CFDictionaryRef)agg, &vyAgg);
		if (st != noErr) { vyAgg = kAudioObjectUnknown; vyStop(); return vyErr("capture du son impossible", st); }
		vyFill = 0;
		st = AudioDeviceCreateIOProcIDWithBlock(&vyProc, vyAgg, NULL,
			^(const AudioTimeStamp *now, const AudioBufferList *in, const AudioTimeStamp *inTime, AudioBufferList *outData, const AudioTimeStamp *outTime) {
				vyFeed(in);
			});
		if (st != noErr) { vyProc = NULL; vyStop(); return vyErr("capture du son impossible", st); }
		st = AudioDeviceStart(vyAgg, vyProc);
		if (st != noErr) { vyStop(); return vyErr("capture du son impossible", st); }
		return NULL;
	}
	return strdup("macOS 14.2 ou plus requis");
}

static unsigned long long vyRead(float *out) {
	os_unfair_lock_lock(&vyLock);
	memcpy(out, vyBands, sizeof vyBands);
	unsigned long long s = vySeq;
	os_unfair_lock_unlock(&vyLock);
	return s;
}
*/
import "C"

import (
	"errors"
	"sync"
	"unsafe"
)

// Bands : nombre de bandes de fréquences, des graves aux aigus.
const Bands = C.VY_BANDS

var mu sync.Mutex

// Available : capture possible sur ce Mac (macOS 14.2 et plus).
func Available() bool { return C.vyAvailable() == 1 }

// Start lance la capture (sans effet si elle tourne déjà).
func Start() error {
	mu.Lock()
	defer mu.Unlock()
	if msg := C.vyStart(); msg != nil {
		defer C.free(unsafe.Pointer(msg))
		return errors.New(C.GoString(msg))
	}
	return nil
}

// Stop arrête la capture et rend la sortie audio intacte.
func Stop() {
	mu.Lock()
	defer mu.Unlock()
	C.vyStop()
}

// Read renvoie la puissance de chaque bande (dB, de -120 à 0 environ) et un numéro qui change à chaque
// nouvelle analyse (≈ 47 par seconde).
func Read() ([]float32, uint64) {
	var b [Bands]C.float
	seq := C.vyRead(&b[0])
	out := make([]float32, Bands)
	for i := range out {
		out[i] = float32(b[i])
	}
	return out, uint64(seq)
}
