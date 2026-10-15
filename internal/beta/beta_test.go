package beta

import (
	"crypto/ed25519"
	"strings"
	"testing"
)

func keys(t *testing.T) (ed25519.PublicKey, ed25519.PrivateKey) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	return pub, priv
}

func TestSignedCodeIsAccepted(t *testing.T) {
	pub, priv := keys(t)
	code := Sign(priv, "Léa")
	if !strings.HasPrefix(code, "VINYL-") {
		t.Fatalf("code %q", code)
	}
	if name, ok := Verify(pub, code); !ok || name != "Léa" {
		t.Errorf("name=%q ok=%v", name, ok)
	}
	// Collé avec des espaces ou un retour à la ligne : toujours valide.
	if name, ok := Verify(pub, "  "+code+"\n"); !ok || name != "Léa" {
		t.Errorf("avec espaces : name=%q ok=%v", name, ok)
	}
}

func TestForgedCodesAreRefused(t *testing.T) {
	pub, priv := keys(t)
	_, other := keys(t)
	good := Sign(priv, "Léa")
	for name, code := range map[string]string{
		"autre clé":    Sign(other, "Léa"),
		"modifié":      flip(good),
		"tronqué":      good[:12],
		"vide":         "",
		"pas un code":  "bonjour",
		"sans préfixe": strings.TrimPrefix(good, "VINYL-"),
	} {
		if _, ok := Verify(pub, code); ok {
			t.Errorf("%s : accepté à tort", name)
		}
	}
}

// Sans clé publique (aucune clé créée), aucun code n'est accepté.
func TestNoKeyAcceptsNothing(t *testing.T) {
	_, priv := keys(t)
	if _, ok := Verify(nil, Sign(priv, "Léa")); ok {
		t.Error("accepté sans clé")
	}
}

// flip change le dernier caractère du code (jamais par lui-même).
func flip(code string) string {
	last := "A"
	if strings.HasSuffix(code, "A") {
		last = "B"
	}
	return code[:len(code)-1] + last
}
