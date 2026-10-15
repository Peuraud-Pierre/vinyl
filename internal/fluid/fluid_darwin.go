//go:build darwin && cgo

// Package fluid lève le bridage de WebKit qui limite les pages à 60 images/s : sur les écrans
// ProMotion (120 Hz), tout macOS suit le trackpad à 120 images/s et l'étagère paraissait saccadée.
// Réglage interne de WebKit (PreferPageRenderingUpdatesNear60FPSEnabled) : chaque appel est vérifié,
// et si une version de macOS le retire, on ne fait simplement rien.
package fluid

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework WebKit
#import <AppKit/AppKit.h>
#import <WebKit/WebKit.h>
#import <objc/message.h>

static NSObject *vyFeature(void) {
	SEL list = NSSelectorFromString(@"_features");
	if (![WKPreferences respondsToSelector:list]) return nil;
	NSArray *all = ((NSArray *(*)(id, SEL))objc_msgSend)([WKPreferences class], list);
	for (NSObject *f in all) {
		if (![f respondsToSelector:NSSelectorFromString(@"key")]) continue;
		if ([[f valueForKey:@"key"] isEqual:@"PreferPageRenderingUpdatesNear60FPSEnabled"]) return f;
	}
	return nil;
}

static int vyAvailable(void) {
	return vyFeature() != nil && [WKPreferences instancesRespondToSelector:NSSelectorFromString(@"_setEnabled:forFeature:")];
}

static WKWebView *vyFindWebView(NSView *v) {
	if ([v isKindOfClass:[WKWebView class]]) return (WKWebView *)v;
	for (NSView *s in v.subviews) {
		WKWebView *w = vyFindWebView(s);
		if (w) return w;
	}
	return nil;
}

static int vyUnlock(void *window) {
	if (window == NULL || !vyAvailable()) return 0;
	WKWebView *web = vyFindWebView(((__bridge NSWindow *)window).contentView);
	if (web == nil) return 0;
	SEL set = NSSelectorFromString(@"_setEnabled:forFeature:");
	((void (*)(id, SEL, BOOL, id))objc_msgSend)(web.configuration.preferences, set, NO, vyFeature());
	return 1;
}
*/
import "C"

import "unsafe"

// Available indique si ce WebKit possède le réglage de bridage à 60 images/s.
func Available() bool { return C.vyAvailable() == 1 }

// Unlock120Hz lève le bridage pour la WKWebView de la fenêtre (NSWindow*). À appeler sur le fil
// principal. Renvoie false si la fenêtre, la vue ou le réglage sont introuvables.
func Unlock120Hz(nsWindow unsafe.Pointer) bool { return C.vyUnlock(nsWindow) == 1 }
