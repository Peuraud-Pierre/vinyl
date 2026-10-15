//go:build darwin && cgo

// Package apps dit si une application est lancée, directement auprès de macOS (NSRunningApplication),
// sans lancer de processus : bien moins coûteux qu'un osascript à chaque sondage.
package apps

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit
#import <AppKit/AppKit.h>
#include <stdlib.h>

static int vyRunning(const char *bundleID) {
	@autoreleasepool {
		NSString *b = [NSString stringWithUTF8String:bundleID];
		return [NSRunningApplication runningApplicationsWithBundleIdentifier:b].count > 0;
	}
}
*/
import "C"

import "unsafe"

// Running indique si une application de cet identifiant (ex. « com.apple.Music ») est lancée.
func Running(bundleID string) bool {
	c := C.CString(bundleID)
	defer C.free(unsafe.Pointer(c))
	return C.vyRunning(c) == 1
}
