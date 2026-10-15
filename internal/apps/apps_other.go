//go:build !darwin || !cgo

package apps

// Running : sans macOS, on ne sait pas ; on suppose l'application lancée.
func Running(string) bool { return true }
