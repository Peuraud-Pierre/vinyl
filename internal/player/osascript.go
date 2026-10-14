package player

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Sep sépare les champs renvoyés par nos scripts (ASCII Unit Separator).
const Sep = "\x1f"

type Runner interface {
	Run(ctx context.Context, script string, args ...string) (string, error)
}

// OSAScript exécute un AppleScript via /usr/bin/osascript.
// Les valeurs utilisateur passent toujours par argv, jamais dans le texte du script.
type OSAScript struct{}

func (OSAScript) Run(ctx context.Context, script string, args ...string) (string, error) {
	// « -- » termine les options : un argument commençant par « - » (titre « -e … ») reste une donnée.
	cmd := exec.CommandContext(ctx, "/usr/bin/osascript", append([]string{"-e", script, "--"}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", classifyError(stderr.String())
	}
	return strings.TrimRight(stdout.String(), "\n"), nil
}

func classifyError(stderr string) error {
	switch {
	case strings.Contains(stderr, "(-1743)"):
		return ErrNotAuthorized
	case strings.Contains(stderr, "(-600)"):
		return ErrNotRunning
	case strings.Contains(stderr, "(-1728)") && strings.Contains(stderr, "application id"):
		// App non installée : le script ne compile pas. Pour nous, c'est comme « non lancée ».
		return ErrNotRunning
	default:
		return fmt.Errorf("osascript: %s", strings.TrimSpace(stderr))
	}
}

func parseFloat(s string) (float64, bool) {
	s = strings.ReplaceAll(strings.TrimSpace(s), ",", ".")
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f < 0 {
		return 0, false
	}
	return f, true
}

// parseSeconds convertit des secondes AppleScript ("213,5" ou "213.5") en millisecondes.
func parseSeconds(s string) int64 {
	f, ok := parseFloat(s)
	if !ok {
		return 0
	}
	return int64(f*1000 + 0.5)
}

// parseMillis lit une valeur déjà en millisecondes.
func parseMillis(s string) int64 {
	f, ok := parseFloat(s)
	if !ok {
		return 0
	}
	return int64(f + 0.5)
}
