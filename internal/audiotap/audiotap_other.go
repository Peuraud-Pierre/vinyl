//go:build !darwin || !cgo

package audiotap

import "errors"

const Bands = 12

func Available() bool           { return false }
func Start() error              { return errors.New("capture du son indisponible sur ce système") }
func Stop()                     {}
func Read() ([]float32, uint64) { return make([]float32, Bands), 0 }
