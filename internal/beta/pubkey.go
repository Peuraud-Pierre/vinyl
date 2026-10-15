package beta

import "crypto/ed25519"

// PublicKey : clé publique des codes contributeur (écrite par cmd/betacode).
var PublicKey = ed25519.PublicKey(nil)
