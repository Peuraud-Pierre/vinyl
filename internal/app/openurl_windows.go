//go:build windows

package app

import "os/exec"

// openURL : gestionnaire de protocoles de Windows (https:, spotify:).
func openURL(u string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Run()
}
