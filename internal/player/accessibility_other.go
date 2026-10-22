//go:build !darwin

package player

// AccessibilityTrusted : sans objet hors Mac.
func AccessibilityTrusted(bool) bool { return true }
