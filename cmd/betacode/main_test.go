package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Première fois : paire créée dans .env (lisible par moi seul) ; ensuite : la même clé est relue.
func TestKeyCreatedOnceInEnvFile(t *testing.T) {
	env := filepath.Join(t.TempDir(), ".env")
	os.WriteFile(env, []byte("AUTRE=1\n"), 0o600)
	priv, created, err := loadOrCreateKey(env, func(string) string { return "" })
	if err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	data, _ := os.ReadFile(env)
	s := string(data)
	pub := base64.StdEncoding.EncodeToString(priv.Public().(ed25519.PublicKey))
	if !strings.Contains(s, "AUTRE=1") || !strings.Contains(s, "VINYL_BETA_PRIVATE_KEY=") || !strings.Contains(s, "VINYL_BETA_PUBLIC_KEY="+pub) {
		t.Errorf(".env : %q", s)
	}
	if fi, _ := os.Stat(env); fi.Mode().Perm() != 0o600 {
		t.Errorf("droits %v", fi.Mode().Perm())
	}
	again, created, err := loadOrCreateKey(env, func(string) string { return "" })
	if err != nil || created || !again.Equal(priv) {
		t.Errorf("relecture : created=%v err=%v", created, err)
	}
}

// Clé fournie par une variable d'environnement (ex. secret) : utilisée, rien n'est écrit.
func TestKeyFromEnvironment(t *testing.T) {
	_, priv, _ := ed25519.GenerateKey(nil)
	env := filepath.Join(t.TempDir(), ".env")
	got, created, err := loadOrCreateKey(env, func(k string) string {
		if k == "VINYL_BETA_PRIVATE_KEY" {
			return base64.StdEncoding.EncodeToString(priv.Seed())
		}
		return ""
	})
	if err != nil || created || !got.Equal(priv) {
		t.Errorf("created=%v err=%v", created, err)
	}
	if _, err := os.Stat(env); !os.IsNotExist(err) {
		t.Error(".env créé à tort")
	}
}
