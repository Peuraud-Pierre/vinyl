package auth

import "testing"

func TestChallengeRFC7636Vector(t *testing.T) {
	// Vecteur de test de l'annexe B de la RFC 7636
	if got := Challenge("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"); got != "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM" {
		t.Errorf("got %s", got)
	}
}

func TestVerifierLengthAndCharset(t *testing.T) {
	v, err := NewVerifier()
	if err != nil {
		t.Fatal(err)
	}
	if len(v) < 43 || len(v) > 128 {
		t.Errorf("longueur %d", len(v))
	}
	for _, c := range v {
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-' || c == '_') {
			t.Fatalf("caractère invalide %q", c)
		}
	}
}
