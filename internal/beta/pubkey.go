package beta

import (
	"crypto/ed25519"
	"encoding/base64"
)

// publicKey : clé publique des codes contributeur (base64), inscrite à la fabrication
// (-ldflags "-X vinyl/internal/beta.publicKey=…") depuis VINYL_BETA_PUBLIC_KEY : le fichier .env en local,
// un secret GitHub pour les releases. Vide : aucun code n'est accepté.
var publicKey string

// PublicKey : clé publique de Vinyl (nil si absente de la fabrication).
var PublicKey = parsePublicKey(publicKey)

func parsePublicKey(s string) ed25519.PublicKey {
	raw, err := base64.StdEncoding.DecodeString(s)
	if err != nil || len(raw) != ed25519.PublicKeySize {
		return nil
	}
	return ed25519.PublicKey(raw)
}
