//go:build darwin

package sysvolume

import (
	"context"
	"strconv"

	"vinyl/internal/player"
)

var run player.Runner = player.OSAScript{}

const getScript = `set s to get volume settings
return ((output volume of s) as string) & " " & ((output muted of s) as string)`

// Entier en argv (pas de conversion texte dépendante de la locale) ; régler le volume rétablit le son.
const setScript = `on run argv
	set volume output volume ((item 1 of argv) as integer) without output muted
end run`

func Get(ctx context.Context) (int, error) {
	out, err := run.Run(ctx, getScript)
	if err != nil {
		return 0, err
	}
	return parseSettings(out)
}

func Set(ctx context.Context, v int) error {
	_, err := run.Run(ctx, setScript, strconv.Itoa(clamp(v)))
	return err
}
