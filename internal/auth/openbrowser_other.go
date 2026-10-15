//go:build !darwin && !windows

package auth

import "os/exec"

func openBrowser(u string) error { return exec.Command("xdg-open", u).Run() }
