//go:build darwin

package app

import (
	"os/exec"
	"strings"
)

// openURL : les liens Apple Music s'ouvrent dans l'app Musique.
func openURL(u string) error {
	if strings.HasPrefix(u, "https://music.apple.com/") {
		u = "music://" + strings.TrimPrefix(u, "https://")
	}
	return exec.Command("open", u).Run()
}
