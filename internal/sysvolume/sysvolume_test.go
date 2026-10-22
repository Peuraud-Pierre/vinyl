package sysvolume

import "testing"

func TestParseSettings(t *testing.T) {
	for in, want := range map[string]int{"31 false": 31, "31 true": 0, "100 false\n": 100, "140 false": 100} {
		if got, err := parseSettings(in); err != nil || got != want {
			t.Errorf("%q : got %d, %v ; want %d", in, got, err, want)
		}
	}
	if _, err := parseSettings("missing value"); err == nil {
		t.Error("sortie invalide acceptée")
	}
}
