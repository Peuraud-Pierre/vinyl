//go:build darwin

package player

/*
#cgo LDFLAGS: -framework ApplicationServices -framework CoreFoundation
#include <ApplicationServices/ApplicationServices.h>

static int vinylAXTrusted(int prompt) {
	const void *keys[] = { kAXTrustedCheckOptionPrompt };
	const void *vals[] = { prompt ? kCFBooleanTrue : kCFBooleanFalse };
	CFDictionaryRef opts = CFDictionaryCreate(NULL, keys, vals, 1,
		&kCFCopyStringDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	Boolean ok = AXIsProcessTrustedWithOptions(opts);
	CFRelease(opts);
	return ok ? 1 : 0;
}
*/
import "C"

// AccessibilityTrusted dit si Vinyl a l'autorisation Accessibilité (nécessaire pour piloter la fenêtre de
// Musique). Avec prompt, macOS affiche sa demande (bouton vers les Réglages Système) si elle manque.
func AccessibilityTrusted(prompt bool) bool {
	p := C.int(0)
	if prompt {
		p = 1
	}
	return C.vinylAXTrusted(p) == 1
}
