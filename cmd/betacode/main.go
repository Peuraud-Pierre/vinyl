// Commande betacode : fabrique un code contributeur (bêtas). Crée la paire de clés au premier usage :
// la clé privée dans ~/.vinyl/beta-private.key (jamais dans le dépôt), la clé publique dans
// internal/beta/pubkey.go (à commiter).
package main

import (
	"crypto/ed25519"
	"fmt"
	"os"
	"path/filepath"

	"vinyl/internal/beta"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage : ./beta-code.sh \"Prénom\"")
		os.Exit(2)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		fail(err)
	}
	keyPath := filepath.Join(home, ".vinyl", "beta-private.key")
	seed, err := os.ReadFile(keyPath)
	if os.IsNotExist(err) {
		_, priv, err := ed25519.GenerateKey(nil)
		if err != nil {
			fail(err)
		}
		seed = priv.Seed()
		if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
			fail(err)
		}
		if err := os.WriteFile(keyPath, seed, 0o600); err != nil {
			fail(err)
		}
		pub := priv.Public().(ed25519.PublicKey)
		src := fmt.Sprintf("package beta\n\nimport \"crypto/ed25519\"\n\n// PublicKey : clé publique des codes contributeur (écrite par cmd/betacode).\nvar PublicKey = ed25519.PublicKey(%#v)\n", []byte(pub))
		if err := os.WriteFile(filepath.Join("internal", "beta", "pubkey.go"), []byte(src), 0o644); err != nil {
			fail(err)
		}
		fmt.Fprintln(os.Stderr, "Nouvelle clé créée : "+keyPath+" (à garder précieusement, jamais dans le dépôt).")
		fmt.Fprintln(os.Stderr, "internal/beta/pubkey.go mis à jour : commite-le et publie une version pour que les codes soient reconnus.")
	} else if err != nil {
		fail(err)
	}
	if len(seed) != ed25519.SeedSize {
		fail(fmt.Errorf("clé invalide : %s", keyPath))
	}
	fmt.Println(beta.Sign(ed25519.NewKeyFromSeed(seed), os.Args[1]))
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
