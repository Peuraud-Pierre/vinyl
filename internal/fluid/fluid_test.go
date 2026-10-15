//go:build darwin && cgo

package fluid

import "testing"

// Le réglage WebKit qui bride les pages à 60 images/s doit exister sur ce Mac : sans lui,
// l'étagère reste à 60 images/s sur les écrans ProMotion (120 Hz) et paraît saccadée au trackpad.
func TestFeatureAvailable(t *testing.T) {
	if !Available() {
		t.Fatal("réglage PreferPageRenderingUpdatesNear60FPSEnabled introuvable dans WebKit")
	}
}

// Sans fenêtre, rien à faire et surtout pas de plantage.
func TestUnlockNilWindow(t *testing.T) {
	if Unlock120Hz(nil) {
		t.Fatal("aucune fenêtre : rien ne doit être débloqué")
	}
}
