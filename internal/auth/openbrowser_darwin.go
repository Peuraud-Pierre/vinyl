//go:build darwin

package auth

import "os/exec"

func openBrowser(u string) error { return exec.Command("open", u).Run() }
