// Commande betacode : fabrique un code contributeur (bêtas). La clé privée vient de la variable
// VINYL_BETA_PRIVATE_KEY ou du fichier .env à la racine du dépôt (ignoré par git) ; au premier usage, la
// paire est créée et inscrite dans .env (VINYL_BETA_PRIVATE_KEY, VINYL_BETA_PUBLIC_KEY).
package main

import (
	"bufio"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"

	"vinyl/internal/beta"
)

const (
	privVar = "VINYL_BETA_PRIVATE_KEY"
	pubVar  = "VINYL_BETA_PUBLIC_KEY"
)

func main() {
	if len(os.Args) != 2 || strings.TrimSpace(os.Args[1]) == "" {
		fmt.Fprintln(os.Stderr, "usage : ./beta-code.sh \"Prénom\"")
		os.Exit(2)
	}
	if _, err := os.Stat("go.mod"); err != nil {
		fail(errors.New("à lancer depuis la racine du dépôt (./beta-code.sh)"))
	}
	priv, created, err := loadOrCreateKey(".env", os.Getenv)
	if err != nil {
		fail(err)
	}
	if created {
		pub := base64.StdEncoding.EncodeToString(priv.Public().(ed25519.PublicKey))
		fmt.Fprintln(os.Stderr, "Nouvelle paire de clés inscrite dans .env (jamais dans le dépôt ; garde une copie de ce fichier).")
		fmt.Fprintln(os.Stderr, "Pour les releases, ajoute sur GitHub (Settings › Secrets and variables › Actions) le secret")
		fmt.Fprintln(os.Stderr, "  "+pubVar+" = "+pub)
		fmt.Fprintln(os.Stderr, "puis publie une version : les codes ne sont reconnus qu'à partir de celle-ci.")
	}
	fmt.Println(beta.Sign(priv, strings.TrimSpace(os.Args[1])))
}

// loadOrCreateKey lit la clé privée (variable d'environnement, sinon fichier .env) ou crée la paire et
// l'ajoute au fichier .env (droits 600). created : la paire vient d'être créée.
func loadOrCreateKey(envPath string, getenv func(string) string) (ed25519.PrivateKey, bool, error) {
	if v := getenv(privVar); v != "" {
		k, err := decodeSeed(v)
		return k, false, err
	}
	if v, err := readEnvFile(envPath, privVar); err != nil {
		return nil, false, err
	} else if v != "" {
		k, err := decodeSeed(v)
		return k, false, err
	}
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		return nil, false, err
	}
	f, err := os.OpenFile(envPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, false, err
	}
	defer f.Close()
	if err := f.Chmod(0o600); err != nil {
		return nil, false, err
	}
	_, err = fmt.Fprintf(f, "\n# Codes contributeur Vinyl (bêtas) — ne jamais commiter ce fichier\n%s=%s\n%s=%s\n",
		privVar, base64.StdEncoding.EncodeToString(priv.Seed()), pubVar, base64.StdEncoding.EncodeToString(pub))
	return priv, err == nil, err
}

func decodeSeed(v string) (ed25519.PrivateKey, error) {
	seed, err := base64.StdEncoding.DecodeString(strings.TrimSpace(v))
	if err != nil || len(seed) != ed25519.SeedSize {
		return nil, fmt.Errorf("%s invalide", privVar)
	}
	return ed25519.NewKeyFromSeed(seed), nil
}

// readEnvFile renvoie la valeur d'une variable d'un fichier .env ("" si absente ou fichier inexistant).
func readEnvFile(path, key string) (string, error) {
	f, err := os.Open(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		k, v, ok := strings.Cut(strings.TrimSpace(sc.Text()), "=")
		if ok && strings.TrimSpace(k) == key {
			return strings.Trim(strings.TrimSpace(v), `"'`), nil
		}
	}
	return "", sc.Err()
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
