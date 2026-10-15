//go:build darwin && cgo

package apps

import "testing"

func TestRunning(t *testing.T) {
	if !Running("com.apple.finder") {
		t.Error("le Finder est toujours lancé")
	}
	if Running("com.exemple.inexistant") {
		t.Error("une application inexistante n'est pas lancée")
	}
}
