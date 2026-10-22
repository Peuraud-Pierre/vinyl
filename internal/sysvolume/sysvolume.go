// Package sysvolume lit et règle le volume de sortie de l'ordinateur (0 à 100), quel que soit le lecteur.
package sysvolume

import (
	"fmt"
	"strconv"
	"strings"
)

func clamp(v int) int { return min(max(v, 0), 100) }

// parseSettings lit « volume muet » (ex. « 31 false ») : un son coupé vaut 0.
func parseSettings(out string) (int, error) {
	f := strings.Fields(out)
	if len(f) != 2 {
		return 0, fmt.Errorf("volume : sortie inattendue %q", out)
	}
	v, err := strconv.Atoi(f[0])
	if err != nil {
		return 0, fmt.Errorf("volume : sortie inattendue %q", out)
	}
	if f[1] == "true" {
		return 0, nil
	}
	return clamp(v), nil
}
