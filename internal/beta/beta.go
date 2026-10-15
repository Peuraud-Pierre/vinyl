// Package beta gère les codes contributeur : un code donne accès aux versions bêta dans l'app. Il est
// signé (Ed25519) avec la clé privée de l'auteur, qui ne quitte pas son Mac ; Vinyl ne contient que la
// clé publique, donc un code ne peut pas être fabriqué en lisant le code de Vinyl.
package beta

import (
	"crypto/ed25519"
	"encoding/base64"
	"strings"
)

const prefix = "VINYL-"

func message(name string) []byte { return []byte("vinyl-beta:" + name) }

// Sign fabrique le code d'un contributeur.
func Sign(priv ed25519.PrivateKey, name string) string {
	sig := ed25519.Sign(priv, message(name))
	return prefix + base64.RawURLEncoding.EncodeToString(append([]byte(name), sig...))
}

// Verify vérifie un code et renvoie le nom du contributeur.
func Verify(pub ed25519.PublicKey, code string) (string, bool) {
	code = strings.Join(strings.Fields(code), "") // espaces et retours à la ligne collés avec le code
	if !strings.HasPrefix(code, prefix) || len(pub) != ed25519.PublicKeySize {
		return "", false
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(code, prefix))
	if err != nil || len(raw) <= ed25519.SignatureSize {
		return "", false
	}
	name, sig := string(raw[:len(raw)-ed25519.SignatureSize]), raw[len(raw)-ed25519.SignatureSize:]
	if !ed25519.Verify(pub, message(name), sig) {
		return "", false
	}
	return name, true
}

// Check vérifie un code avec la clé publique de Vinyl.
func Check(code string) (string, bool) { return Verify(PublicKey, code) }
