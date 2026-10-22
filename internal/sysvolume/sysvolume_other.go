//go:build !darwin && !windows

package sysvolume

import (
	"context"
	"errors"
)

var errUnsupported = errors.New("volume du système non pris en charge")

func Get(context.Context) (int, error) { return 0, errUnsupported }
func Set(context.Context, int) error   { return errUnsupported }
