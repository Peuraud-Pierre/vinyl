//go:build windows

package auth

import "os/exec"

func openBrowser(u string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Run()
}
