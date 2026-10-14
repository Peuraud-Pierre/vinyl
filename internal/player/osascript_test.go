package player

import (
	"context"
	"errors"
	"testing"
)

func TestParseSecondsAcceptsCommaAndDot(t *testing.T) {
	cases := map[string]int64{
		"213.5":         213500,
		"213,5":         213500,
		"0":             0,
		"":              0,
		"missing value": 0,
		" 12,25 ":       12250,
		"abc":           0,
		"-3":            0,
	}
	for in, want := range cases {
		if got := parseSeconds(in); got != want {
			t.Errorf("parseSeconds(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestParseMillis(t *testing.T) {
	cases := map[string]int64{"185000": 185000, "185000,0": 185000, "": 0, "missing value": 0}
	for in, want := range cases {
		if got := parseMillis(in); got != want {
			t.Errorf("parseMillis(%q) = %d, want %d", in, got, want)
		}
	}
}

func TestClassifyError(t *testing.T) {
	if !errors.Is(classifyError("execution error: Not authorized to send Apple events to Music. (-1743)"), ErrNotAuthorized) {
		t.Error("-1743 doit donner ErrNotAuthorized")
	}
	if !errors.Is(classifyError("execution error: Application isn't running. (-600)"), ErrNotRunning) {
		t.Error("-600 doit donner ErrNotRunning")
	}
	// App non installée : AppleScript ne peut même pas compiler le « tell ».
	if !errors.Is(classifyError(`84:119: syntax error: Can’t get application id "com.spotify.client". (-1728)`), ErrNotRunning) {
		t.Error("app absente (-1728 sur application id) doit donner ErrNotRunning")
	}
	err := classifyError("syntax error")
	if err == nil || errors.Is(err, ErrNotAuthorized) || errors.Is(err, ErrNotRunning) {
		t.Errorf("erreur générique attendue, obtenu %v", err)
	}
}

func TestOSAScriptPassesArgsWithoutInterpolation(t *testing.T) {
	if testing.Short() {
		t.Skip("appelle /usr/bin/osascript")
	}
	title := `Don't Stop "Live" – Beyoncé 🎵`
	out, err := OSAScript{}.Run(context.Background(), "on run argv\nreturn item 1 of argv\nend run", title)
	if err != nil {
		t.Fatal(err)
	}
	if out != title {
		t.Errorf("got %q, want %q", out, title)
	}
}

func TestOSAScriptArgStartingWithDashIsNotAnOption(t *testing.T) {
	if testing.Short() {
		t.Skip("appelle /usr/bin/osascript")
	}
	// Un titre « -e … » ne doit jamais être lu comme une option d'osascript (injection de code).
	for _, arg := range []string{"-e", "-ism", `property p : do shell script "exit 1"`} {
		out, err := OSAScript{}.Run(context.Background(), "on run argv\nreturn (count of argv) as string\nend run", arg, "x")
		if err != nil || out != "2" {
			t.Errorf("arg %q : out=%q err=%v (attendu 2 arguments)", arg, out, err)
		}
	}
}
