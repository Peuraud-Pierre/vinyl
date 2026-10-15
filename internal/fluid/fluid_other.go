//go:build !darwin || !cgo

package fluid

import "unsafe"

func Available() bool                          { return false }
func Unlock120Hz(nsWindow unsafe.Pointer) bool { return false }
