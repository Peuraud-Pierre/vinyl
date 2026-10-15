# Mises à jour de Vinyl dans l'app — plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Vinyl propose les nouvelles versions publiées sur GitHub, les télécharge, les installe et redémarre ; les bêtas ne sont proposées qu'avec un code contributeur signé.

**Architecture:** Le système de mise à jour de Wails v3 (`app.Updater`, `pkg/updater`) fait le travail délicat (vérification, téléchargement vérifié, remplacement, redémarrage). Vinyl lui donne sa version (injectée à la fabrication), une source GitHub à deux canaux (`internal/update`), un code contributeur vérifié par signature Ed25519 (`internal/beta`), et son interface (bandeau + Réglages).

**Tech Stack:** Go 1.26, Wails v3.0.0-beta.29 (`pkg/updater`, `pkg/updater/providers/github`), `crypto/ed25519`, TypeScript + Vitest, GitHub Actions.

**Spec:** `docs/superpowers/specs/2026-10-15-vinyl-mises-a-jour-design.md`

## Global Constraints

- Branche : `feat/mises-a-jour`, créée depuis `main`.
- Aucune mention de Claude dans les commits (pas de `Co-Authored-By`) ; messages `type(portée): description` en français.
- Tests Go Mac : `CGO_ENABLED=1 go test -race . ./internal/...` ; `vet` : `CGO_ENABLED=1 go vet . ./internal/...` (jamais `./...`).
- Vérification Windows depuis le Mac : `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/vinyl.exe .` et `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet . ./internal/...`.
- Frontend : `cd frontend && npx tsc --noEmit && npm test`.
- Liaisons après tout changement de méthode exportée du service : `PATH="$PATH:$(go env GOPATH)/bin" wails3 generate bindings -clean=true -ts -i`.
- Dépôt GitHub : `Peuraud-Pierre/vinyl` ; fichier d'empreintes : `SHA256SUMS` ; fichiers de mise à jour : `Vinyl-darwin-arm64.zip`, `Vinyl-windows-amd64.zip` ; installateur : `Vinyl-Setup.exe`.
- Vérification toutes les 6 h ; version « dev » = aucune vérification.
- La clé privée des codes contributeur ne doit jamais entrer dans le dépôt (`~/.vinyl/beta-private.key`).
- Textes de l'interface en français.

## Review Focus

1. Pas de réseau ou GitHub indisponible au lancement : rien ne s'affiche et l'app démarre normalement (pas d'erreur visible, nouvel essai plus tard). Attention : Wails émet aussi `wails:updater:error` pour un échec de vérification ; le bandeau ne doit l'afficher que pendant une installation — tests Task 5 et Task 6.
2. Code contributeur collé avec des espaces ou un retour à la ligne : il doit être accepté s'il est valide — test Task 2.
3. Code contributeur retiré : la bêta déjà proposée ne doit plus l'être à la vérification suivante — test Task 4.
4. Clic répété sur « Mettre à jour » pendant un téléchargement : pas de second téléchargement ni d'erreur affichée — test Task 5.
5. Version « dev » (fabriquée à la main) : aucune vérification, aucun bandeau, et Réglages affiche « Vinyl (développement) » — test Task 5.

---

### Task 1: Version de l'app

**Files:**
- Modify: `main.go` (variable `version`), `internal/app/service.go` (champ + `SetVersion`, `Version`), `build/darwin/Taskfile.yml:45`, `build/windows/Taskfile.yml:60`
- Test: `internal/app/update_test.go`

**Interfaces:**
- Produces: `var version = "dev"` (package main, remplacée par `-X main.version=<x>`), `func (s *VinylService) SetVersion(v string)`, `func (s *VinylService) Version() string`.

- [ ] **Step 1: Write the failing test** — `internal/app/update_test.go`

```go
package app

import (
	"testing"
	"time"

	"vinyl/internal/player"
)

func newBare() *VinylService {
	return New(nil, player.NewWatcher(nil, func() string { return "auto" }, func(player.Snapshot) {}, time.Now), nil)
}

func TestVersion(t *testing.T) {
	s := newBare()
	if v := s.Version(); v != "dev" {
		t.Errorf("par défaut : %q", v)
	}
	s.SetVersion("1.0.3")
	if v := s.Version(); v != "1.0.3" {
		t.Errorf("injectée : %q", v)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=1 go test ./internal/app/ -run TestVersion`
Expected: FAIL (`Version` / `SetVersion` absents).

- [ ] **Step 3: Implement**

`service.go` : champ `version string` ; dans `New`, `version: "dev"` ;

```go
// SetVersion : numéro de la release (sans « v »), « dev » pour une fabrication à la main.
func (s *VinylService) SetVersion(v string) {
	if v != "" {
		s.version = v
	}
}

// Version : numéro de version de l'app (affiché dans Réglages).
func (s *VinylService) Version() string { return s.version }
```

`main.go` : sous `var projectDir string`, ajouter

```go
// version est le numéro de la release, inscrit à la fabrication (-ldflags "-X main.version=1.0.3").
var version = "dev"
```
et après la création du service : `service.SetVersion(version)`.

`build/darwin/Taskfile.yml:45` : dans `-ldflags="…-X ''main.projectDir={{.ROOT_DIR}}''"`, ajouter ` -X ''main.version={{.VINYL_VERSION | default "dev"}}''` après `main.projectDir=…''`.

`build/windows/Taskfile.yml:60` : dans la partie production de `BUILD_FLAGS`, remplacer `-ldflags="-w -s -H windowsgui"` par `-ldflags="-w -s -H windowsgui -X main.version={{.VINYL_VERSION | default "dev"}}"`.

Régénérer les liaisons.

- [ ] **Step 4: Run tests and builds**

Run: `CGO_ENABLED=1 go test -race . ./internal/... && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/vinyl.exe . && CGO_ENABLED=1 go build -ldflags "-X main.version=9.9.9" -o /tmp/vinyl-v . && strings /tmp/vinyl-v | grep -c 9.9.9`
Expected: tests verts ; `1` (ou plus) pour la dernière commande.

- [ ] **Step 5: Commit**

```bash
git add main.go internal/app build/darwin/Taskfile.yml build/windows/Taskfile.yml frontend/bindings
git commit -m "feat(version): numéro de version inscrit à la fabrication"
```

---

### Task 2: Code contributeur signé (`internal/beta`)

**Files:**
- Create: `internal/beta/beta.go`, `internal/beta/pubkey.go`, `cmd/betacode/main.go`, `beta-code.sh`
- Test: `internal/beta/beta_test.go`

**Interfaces:**
- Produces: `func Sign(priv ed25519.PrivateKey, name string) string` (code `VINYL-…`), `func Verify(pub ed25519.PublicKey, code string) (name string, ok bool)`, `func Check(code string) (string, bool)` (avec la clé publique de Vinyl), `var PublicKey ed25519.PublicKey` (dans `pubkey.go`).

- [ ] **Step 1: Write the failing test** — `internal/beta/beta_test.go`

```go
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
		"autre clé":     Sign(other, "Léa"),
		"modifié":       good[:len(good)-2] + "AA",
		"tronqué":       good[:12],
		"vide":          "",
		"pas un code":   "bonjour",
		"sans préfixe":  strings.TrimPrefix(good, "VINYL-"),
	} {
		if _, ok := Verify(pub, code); ok {
			t.Errorf("%s : accepté à tort", name)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/beta/`
Expected: FAIL (paquet sans code).

- [ ] **Step 3: Implement** — `internal/beta/beta.go`

```go
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
	code = strings.TrimSpace(code)
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
```

`internal/beta/pubkey.go` (régénéré par `cmd/betacode` à la première utilisation ; vide tant qu'aucune clé n'existe, donc aucun code accepté) :

```go
package beta

import "crypto/ed25519"

// PublicKey : clé publique des codes contributeur (écrite par cmd/betacode).
var PublicKey = ed25519.PublicKey(nil)
```

`cmd/betacode/main.go` :

```go
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
```

`beta-code.sh` (exécutable) :

```bash
#!/bin/bash
# Fabrique un code contributeur (accès aux bêtas dans Vinyl) : ./beta-code.sh "Prénom"
set -euo pipefail
cd "$(dirname "$0")"
go run ./cmd/betacode "$@"
```

- [ ] **Step 4: Run tests**

Run: `go test -race ./internal/beta/ && go vet ./cmd/betacode/ ./internal/beta/ && chmod +x beta-code.sh && bash -n beta-code.sh`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/beta cmd/betacode beta-code.sh
git commit -m "feat(bêta): codes contributeur signés et script beta-code.sh"
```

---

### Task 3: Code contributeur dans les réglages

**Files:**
- Modify: `internal/settings/settings.go` (`Settings.BetaCode`, `Patch.BetaCode`, `Update`)
- Test: `internal/settings/settings_test.go`

**Interfaces:**
- Produces: `Settings.BetaCode string` (json `betaCode`), `Patch.BetaCode *string`.

- [ ] **Step 1: Write the failing test** — ajout dans `internal/settings/settings_test.go`

```go
func TestBetaCodeIsSaved(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "s.json"))
	s.Load()
	code := "VINYL-abc"
	if _, err := s.Update(Patch{BetaCode: &code}); err != nil {
		t.Fatal(err)
	}
	r := NewStore(s.path)
	r.Load()
	if r.Get().BetaCode != code {
		t.Errorf("relu : %q", r.Get().BetaCode)
	}
}
```
(adapter `NewStore`/`s.path` aux noms réels du paquet en lisant `settings_test.go` existant.)

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/settings/`
Expected: FAIL (`BetaCode` absent).

- [ ] **Step 3: Implement**

`Settings` : `BetaCode string \`json:"betaCode,omitempty"\` // code contributeur (bêtas)` ; `Patch` : `BetaCode *string \`json:"betaCode,omitempty"\`` ; dans `Update` : `if p.BetaCode != nil { v.BetaCode = strings.TrimSpace(*p.BetaCode) }`.

- [ ] **Step 4: Run tests**

Run: `go test -race ./internal/settings/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/settings
git commit -m "feat(réglages): code contributeur enregistré"
```

---

### Task 4: Source de mise à jour à deux canaux (`internal/update`)

**Files:**
- Create: `internal/update/channel.go`
- Test: `internal/update/channel_test.go`

**Interfaces:**
- Consumes: `updater.Provider`, `updater.CheckRequest`, `updater.Release` (Wails).
- Produces: `func NewChannel(stable, beta updater.Provider, betaOn func() bool) *Channel` (implémente `updater.Provider`, `Name() == "vinyl-github"`), `func NewGitHub(betaOn func() bool) (*Channel, error)` (deux sources GitHub `Peuraud-Pierre/vinyl`, `SHA256SUMS`), `const AssetMac = "Vinyl-darwin-arm64.zip"`, `const AssetWindows = "Vinyl-windows-amd64.zip"`.

- [ ] **Step 1: Write the failing test** — `internal/update/channel_test.go`

```go
package update

import (
	"context"
	"io"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

type fakeProvider struct {
	name      string
	version   string
	downloads int
}

func (f *fakeProvider) Name() string { return f.name }
func (f *fakeProvider) Check(context.Context, updater.CheckRequest) (*updater.Release, error) {
	return &updater.Release{Version: f.version}, nil
}
func (f *fakeProvider) Download(context.Context, *updater.Release, io.Writer, func(int64, int64)) error {
	f.downloads++
	return nil
}

// Sans code contributeur : versions stables seulement ; avec : bêtas comprises ; code retiré : retour au stable.
func TestChannelFollowsBetaAccess(t *testing.T) {
	stable, beta := &fakeProvider{name: "s", version: "1.0.3"}, &fakeProvider{name: "b", version: "1.1.0-beta.1"}
	on := false
	c := NewChannel(stable, beta, func() bool { return on })
	ctx := context.Background()
	if r, _ := c.Check(ctx, updater.CheckRequest{}); r.Version != "1.0.3" {
		t.Errorf("sans code : %q", r.Version)
	}
	on = true
	r, _ := c.Check(ctx, updater.CheckRequest{})
	if r.Version != "1.1.0-beta.1" {
		t.Fatalf("avec code : %q", r.Version)
	}
	c.Download(ctx, r, io.Discard, func(int64, int64) {})
	if beta.downloads != 1 || stable.downloads != 0 {
		t.Errorf("téléchargement par la source qui a proposé la version")
	}
	on = false
	if r, _ := c.Check(ctx, updater.CheckRequest{}); r.Version != "1.0.3" {
		t.Errorf("code retiré : %q", r.Version)
	}
}

// Les noms des fichiers publiés sont bien ceux que Wails choisit pour chaque système.
func TestAssetNamesMatchWailsMatcher(t *testing.T) {
	assets := []github.ReleaseAsset{{Name: "Vinyl-Setup.exe"}, {Name: "SHA256SUMS"}, {Name: AssetMac}, {Name: AssetWindows}}
	for req, want := range map[updater.CheckRequest]string{
		{Platform: "darwin", Arch: "arm64"}:  AssetMac,
		{Platform: "windows", Arch: "amd64"}: AssetWindows,
	} {
		i := github.DefaultAssetMatcher(req, assets)
		if i < 0 || assets[i].Name != want {
			t.Errorf("%s/%s : fichier choisi %d", req.Platform, req.Arch, i)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/update/`
Expected: FAIL (paquet sans code).

- [ ] **Step 3: Implement** — `internal/update/channel.go`

```go
// Package update relie Vinyl au système de mise à jour de Wails : une source GitHub à deux canaux,
// stable pour tous, bêta pour les contributeurs (code valide).
package update

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/wailsapp/wails/v3/pkg/updater"
	"github.com/wailsapp/wails/v3/pkg/updater/providers/github"
)

const (
	Repository   = "Peuraud-Pierre/vinyl"
	Checksums    = "SHA256SUMS"
	AssetMac     = "Vinyl-darwin-arm64.zip"
	AssetWindows = "Vinyl-windows-amd64.zip"
)

// Channel choisit, à chaque vérification, la source stable ou celle qui inclut les bêtas.
type Channel struct {
	stable, beta updater.Provider
	betaOn       func() bool
	mu           sync.Mutex
	last         updater.Provider // source de la dernière version proposée (pour la télécharger)
}

func NewChannel(stable, beta updater.Provider, betaOn func() bool) *Channel {
	return &Channel{stable: stable, beta: beta, betaOn: betaOn}
}

// NewGitHub : releases GitHub de Vinyl, empreintes vérifiées avec SHA256SUMS.
func NewGitHub(betaOn func() bool) (*Channel, error) {
	stable, err := github.New(github.Config{Repository: Repository, ChecksumAsset: Checksums})
	if err != nil {
		return nil, err
	}
	beta, err := github.New(github.Config{Repository: Repository, ChecksumAsset: Checksums, Prerelease: true})
	if err != nil {
		return nil, err
	}
	return NewChannel(stable, beta, betaOn), nil
}

func (c *Channel) Name() string { return "vinyl-github" }

func (c *Channel) Check(ctx context.Context, req updater.CheckRequest) (*updater.Release, error) {
	p := c.stable
	if c.betaOn() {
		p = c.beta
	}
	c.mu.Lock()
	c.last = p
	c.mu.Unlock()
	return p.Check(ctx, req)
}

func (c *Channel) Download(ctx context.Context, r *updater.Release, dst io.Writer, onProgress func(int64, int64)) error {
	c.mu.Lock()
	p := c.last
	c.mu.Unlock()
	if p == nil {
		return errors.New("update: aucune version proposée")
	}
	return p.Download(ctx, r, dst, onProgress)
}
```

- [ ] **Step 4: Run tests**

Run: `go test -race ./internal/update/ && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./internal/update/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/update go.mod go.sum
git commit -m "feat(mises à jour): source GitHub à deux canaux (stable, bêta)"
```

---

### Task 5: Service et démarrage

**Files:**
- Modify: `internal/app/service.go` (méthodes de mise à jour et code contributeur), `main.go` (initialisation de `app.Updater`)
- Test: `internal/app/update_test.go`

**Interfaces:**
- Consumes: Task 1 (`version`), Task 2 (`beta.Check`), Task 3 (`Settings.BetaCode`), Task 4 (`update.NewGitHub`).
- Produces: interface `Updater { Check(ctx) (*updater.Release, error); DownloadAndInstall(ctx) error; Restart(ctx) error; State() updater.State }` (satisfaite par `*updater.Updater`), `func (s *VinylService) SetUpdater(u Updater)`, méthodes liées : `CheckForUpdate() (UpdateInfo, error)`, `InstallUpdate() error`, `BetaAccess() string` (nom du contributeur, "" sinon), `SetBetaCode(code string) (string, error)`, `ReleaseURL(version string) string`. `type UpdateInfo struct { Available bool; Version, Notes string; Dev bool }`.

- [ ] **Step 1: Write the failing tests** — ajout dans `internal/app/update_test.go`

```go
type fakeUpdater struct {
	rel       *updater.Release
	checkErr  error
	installs  int
	restarts  int
	busy      bool
}

func (f *fakeUpdater) Check(context.Context) (*updater.Release, error) { return f.rel, f.checkErr }
func (f *fakeUpdater) DownloadAndInstall(context.Context) error {
	if f.busy {
		return updater.ErrDownloadInProgress
	}
	f.installs++
	return nil
}
func (f *fakeUpdater) Restart(context.Context) error { f.restarts++; return nil }
func (f *fakeUpdater) State() updater.State          { return updater.StateIdle }

func TestDevBuildNeverChecks(t *testing.T) {
	s := newBare()
	s.SetUpdater(&fakeUpdater{rel: &updater.Release{Version: "9.9.9"}})
	info, err := s.CheckForUpdate()
	if err != nil || info.Available || !info.Dev {
		t.Errorf("dev : %+v %v", info, err)
	}
}

func TestUpdateAvailableThenInstalled(t *testing.T) {
	s := newBare()
	s.SetVersion("1.0.2")
	u := &fakeUpdater{rel: &updater.Release{Version: "1.0.3", Notes: "Nouveautés"}}
	s.SetUpdater(u)
	info, err := s.CheckForUpdate()
	if err != nil || !info.Available || info.Version != "1.0.3" {
		t.Fatalf("%+v %v", info, err)
	}
	if err := s.InstallUpdate(); err != nil || u.installs != 1 || u.restarts != 1 {
		t.Errorf("installs=%d restarts=%d err=%v", u.installs, u.restarts, err)
	}
}

// Pas de réseau : pas d'erreur montrée, simplement « rien de nouveau ».
func TestCheckWithoutNetworkIsQuiet(t *testing.T) {
	s := newBare()
	s.SetVersion("1.0.2")
	s.SetUpdater(&fakeUpdater{checkErr: errors.New("dial tcp: no route")})
	if info, err := s.CheckForUpdate(); err != nil || info.Available {
		t.Errorf("%+v %v", info, err)
	}
}

// Second clic pendant un téléchargement : ignoré, sans erreur.
func TestInstallWhileDownloadingIsIgnored(t *testing.T) {
	s := newBare()
	s.SetVersion("1.0.2")
	s.SetUpdater(&fakeUpdater{busy: true})
	if err := s.InstallUpdate(); err != nil {
		t.Errorf("err = %v", err)
	}
}

func TestReleaseURL(t *testing.T) {
	if u := newBare().ReleaseURL("1.0.3"); u != "https://github.com/Peuraud-Pierre/vinyl/releases/tag/v1.0.3" {
		t.Errorf("%q", u)
	}
}
```

Et pour le code contributeur (le service a besoin d'un `settings.Store`) :

```go
func TestBetaCode(t *testing.T) {
	st := settings.NewStore(filepath.Join(t.TempDir(), "s.json"))
	st.Load()
	s := New(st, player.NewWatcher(nil, func() string { return "auto" }, func(player.Snapshot) {}, time.Now), nil)
	pub, priv, _ := ed25519.GenerateKey(nil)
	old := beta.PublicKey
	beta.PublicKey = pub
	defer func() { beta.PublicKey = old }()
	if _, err := s.SetBetaCode("faux"); err == nil {
		t.Error("code invalide accepté")
	}
	name, err := s.SetBetaCode(beta.Sign(priv, "Léa"))
	if err != nil || name != "Léa" || s.BetaAccess() != "Léa" {
		t.Errorf("name=%q err=%v accès=%q", name, err, s.BetaAccess())
	}
	if _, err := s.SetBetaCode(""); err != nil || s.BetaAccess() != "" {
		t.Errorf("retrait : %v %q", err, s.BetaAccess())
	}
}
```
(imports : `context`, `crypto/ed25519`, `errors`, `path/filepath`, `github.com/wailsapp/wails/v3/pkg/updater`, `vinyl/internal/beta`, `vinyl/internal/settings`.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `CGO_ENABLED=1 go test ./internal/app/`
Expected: FAIL (méthodes absentes).

- [ ] **Step 3: Implement** — `internal/app/service.go`

```go
// Updater : ce dont le service a besoin du système de mise à jour de Wails (*updater.Updater).
type Updater interface {
	Check(ctx context.Context) (*updater.Release, error)
	DownloadAndInstall(ctx context.Context) error
	Restart(ctx context.Context) error
	State() updater.State
}

// UpdateInfo : résultat d'une recherche de mise à jour.
type UpdateInfo struct {
	Available bool   `json:"available"`
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	Dev       bool   `json:"dev"` // fabrication à la main : pas de mise à jour
}

func (s *VinylService) SetUpdater(u Updater) { s.updater = u }

// CheckForUpdate cherche une version plus récente (stable, ou bêta avec un code contributeur).
// Sans réseau : rien de nouveau, sans erreur.
func (s *VinylService) CheckForUpdate() (UpdateInfo, error) {
	if s.version == "dev" || s.updater == nil {
		return UpdateInfo{Dev: s.version == "dev"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	rel, err := s.updater.Check(ctx)
	if err != nil || rel == nil {
		return UpdateInfo{}, nil
	}
	return UpdateInfo{Available: true, Version: rel.Version, Notes: rel.Notes}, nil
}

// InstallUpdate télécharge et vérifie la version proposée, la met en place et redémarre Vinyl.
func (s *VinylService) InstallUpdate() error {
	if s.updater == nil {
		return errors.New("mise à jour indisponible")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	if err := s.updater.DownloadAndInstall(ctx); err != nil {
		if errors.Is(err, updater.ErrDownloadInProgress) {
			return nil // déjà en cours
		}
		return err
	}
	return s.updater.Restart(ctx)
}

// ReleaseURL : page GitHub d'une version (repli si l'installation échoue).
func (s *VinylService) ReleaseURL(version string) string {
	return "https://github.com/Peuraud-Pierre/vinyl/releases/tag/v" + strings.TrimPrefix(version, "v")
}

// BetaAccess : nom du contributeur si un code valide est enregistré, "" sinon.
func (s *VinylService) BetaAccess() string {
	if s.store == nil {
		return ""
	}
	name, _ := beta.Check(s.store.Get().BetaCode)
	return name
}

// SetBetaCode enregistre un code contributeur valide (vide : retire l'accès aux bêtas).
func (s *VinylService) SetBetaCode(code string) (string, error) {
	code = strings.TrimSpace(code)
	name := ""
	if code != "" {
		var ok bool
		if name, ok = beta.Check(code); !ok {
			return "", errors.New("code invalide")
		}
	}
	if _, err := s.store.Update(settings.Patch{BetaCode: &code}); err != nil {
		return "", err
	}
	return name, nil
}
```
(champ `updater Updater` dans `VinylService` ; imports `github.com/wailsapp/wails/v3/pkg/updater` et `vinyl/internal/beta`.)

`main.go`, après `wailsApp = application.New(...)` et la création des fenêtres :

```go
	// Mises à jour depuis les releases GitHub (aucune pour une fabrication « dev »).
	if version != "dev" {
		if src, err := update.NewGitHub(func() bool { return service.BetaAccess() != "" }); err == nil {
			if err := wailsApp.Updater.Init(updater.Config{
				CurrentVersion: version, Providers: []updater.Provider{src},
				CheckInterval: 6 * time.Hour, Window: updater.WindowNone,
			}); err == nil {
				service.SetUpdater(wailsApp.Updater)
			} else {
				log.Printf("mises à jour indisponibles : %v", err)
			}
		}
	}
```
(imports `github.com/wailsapp/wails/v3/pkg/updater`, `vinyl/internal/update`.) Régénérer les liaisons.

- [ ] **Step 4: Run tests and builds**

Run: `CGO_ENABLED=1 go test -race . ./internal/... && CGO_ENABLED=1 go vet . ./internal/... && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/vinyl.exe .`
Expected: tout passe.

- [ ] **Step 5: Commit**

```bash
git add internal/app main.go frontend/bindings go.mod go.sum
git commit -m "feat(mises à jour): recherche, installation et code contributeur côté service"
```

---

### Task 6: Interface (bandeau et Réglages)

**Files:**
- Create: `frontend/src/ui/updatebanner.ts`
- Modify: `frontend/src/main.ts` (montage du bandeau), `frontend/index.html` (conteneur `#app-update`), `frontend/src/views/reglages.ts` (Version, Contributeur)
- Test: `frontend/src/ui/updatebanner.test.ts`

**Interfaces:**
- Consumes: Task 5 (`CheckForUpdate`, `InstallUpdate`, `ReleaseURL`, `Version`, `BetaAccess`, `SetBetaCode`), événements Wails `wails:updater:download-progress` (`{written,total}` ou équivalent), `wails:updater:error`.
- Produces: `mountUpdateBanner(el: HTMLElement, deps: BannerDeps): { show(info: UpdateInfo): void; progress(written: number, total: number): void; installFailed(): void }` (`installFailed` ne fait rien hors installation).

- [ ] **Step 1: Write the failing test** — `frontend/src/ui/updatebanner.test.ts`

```ts
import { expect, test, vi } from "vitest";
import { mountUpdateBanner } from "./updatebanner";

const deps = () => ({ install: vi.fn(() => Promise.resolve()), openRelease: vi.fn() });

test("nouvelle version : bandeau et bouton Mettre à jour", async () => {
  const el = document.createElement("div");
  const d = deps();
  const b = mountUpdateBanner(el, d);
  b.show({ available: true, version: "1.0.3", notes: "", dev: false });
  expect(el.hidden).toBe(false);
  expect(el.textContent).toContain("Vinyl 1.0.3 est disponible");
  el.querySelector<HTMLButtonElement>("[data-update]")!.click();
  expect(d.install).toHaveBeenCalledTimes(1);
});

test("rien de nouveau : pas de bandeau", () => {
  const el = document.createElement("div");
  mountUpdateBanner(el, deps()).show({ available: false, version: "", notes: "", dev: false });
  expect(el.hidden).toBe(true);
});

test("téléchargement : pourcentage ; échec : lien vers la release", () => {
  const el = document.createElement("div");
  const d = deps();
  const b = mountUpdateBanner(el, d);
  b.show({ available: true, version: "1.0.3", notes: "", dev: false });
  el.querySelector<HTMLButtonElement>("[data-update]")!.click();
  b.progress(42, 100);
  expect(el.textContent).toContain("42 %");
  b.installFailed();
  expect(el.textContent).toContain("La mise à jour a échoué");
  el.querySelector<HTMLButtonElement>("[data-release]")!.click();
  expect(d.openRelease).toHaveBeenCalledWith("1.0.3");
});

// Wails émet aussi « error » quand une vérification échoue (pas de réseau) : rien ne s'affiche.
test("erreur hors installation : pas de bandeau", () => {
  const el = document.createElement("div");
  mountUpdateBanner(el, deps()).installFailed();
  expect(el.hidden).toBe(true);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/ui/updatebanner.test.ts`
Expected: FAIL (module absent).

- [ ] **Step 3: Implement** — `frontend/src/ui/updatebanner.ts`

```ts
// Bandeau de mise à jour : « Vinyl 1.0.3 est disponible » → téléchargement → redémarrage ; en cas
// d'échec, lien vers la page de la release.
export interface UpdateInfo { available: boolean; version: string; notes: string; dev: boolean; }
export interface BannerDeps { install(): Promise<unknown>; openRelease(version: string): void; }

export function mountUpdateBanner(el: HTMLElement, deps: BannerDeps) {
  let version = "";
  let installing = false;
  el.hidden = true;
  const render = (html: string) => { el.innerHTML = `<div class="glass flex items-center gap-3 rounded-full px-4 py-2 text-[13px]">${html}</div>`; el.hidden = false; };
  const fail = () => {
    installing = false;
    render(`<span>La mise à jour a échoué.</span><button data-release class="font-semibold underline">Voir la release</button>`);
    el.querySelector<HTMLButtonElement>("[data-release]")!.onclick = () => deps.openRelease(version);
  };
  return {
    show(info: UpdateInfo) {
      if (!info.available) { el.hidden = true; return; }
      version = info.version;
      render(`<span>Vinyl ${info.version} est disponible</span><button data-update class="btn-dark !px-3 !py-1 text-[13px]">Mettre à jour</button>`);
      el.querySelector<HTMLButtonElement>("[data-update]")!.onclick = () => {
        installing = true;
        render(`<span>Téléchargement…</span>`);
        deps.install().catch(fail);
      };
    },
    progress(written: number, total: number) {
      if (installing && total > 0) render(`<span>Téléchargement… ${Math.round((written / total) * 100)} %</span>`);
    },
    installFailed() { if (installing) fail(); },
  };
}
```

`frontend/index.html` : à côté de `#widget-update`, ajouter `<div id="app-update" class="no-drag fixed right-40 top-3 z-50" hidden></div>`.

`main.ts` :

```ts
// Mises à jour : vérification au lancement, puis au fil des événements du système de mise à jour.
const banner = mountUpdateBanner(document.getElementById("app-update")!, {
  install: () => api.InstallUpdate(),
  openRelease: (v) => void api.ReleaseURL(v).then((u) => api.OpenURL(u)),
});
api.CheckForUpdate().then((i) => banner.show(i)).catch(() => {});
Events.On("wails:updater:update-available", () => api.CheckForUpdate().then((i) => banner.show(i)).catch(() => {}));
Events.On("wails:updater:download-progress", (e) => {
  const p = e.data as { written: number; total: number }; // updater.Progress
  banner.progress(p.written, p.total);
});
Events.On("wails:updater:error", () => banner.installFailed());
```
(`Events` vient de `@wailsio/runtime` ; vérifier dans `pkg/updater/events.go` le nom exact des événements.)

`reglages.ts` : nouvelle section « Version » :

```ts
// Version et mises à jour.
const vSection = el.querySelector<HTMLElement>("#version-section")!;
const renderVersion = async () => {
  const [v, who] = await Promise.all([api.Version(), api.BetaAccess()]);
  vSection.innerHTML = `
    <h2>Version</h2>
    <div class="flex items-center gap-3">
      <span>${v === "dev" ? "Vinyl (développement)" : `Vinyl ${escapeAttr(v)}`}</span>
      ${v === "dev" ? "" : `<button id="up-check" class="btn-dark cursor-pointer">Rechercher une mise à jour</button>`}
      <span id="up-msg" class="text-[14px]"></span>
    </div>
    <details class="text-[14px]" ${who ? "open" : ""}>
      <summary class="cursor-pointer">Contributeur</summary>
      ${who
        ? `<p class="mt-2">Bêtas activées pour ${escapeAttr(who)}. <button id="beta-off" class="underline">Retirer</button></p>`
        : `<div class="mt-2 flex gap-2"><input id="beta-code" class="field flex-1" placeholder="Code contributeur"><button id="beta-on" class="btn-dark">Activer</button></div><p id="beta-msg" class="mt-1"></p>`}
    </details>`;
  vSection.querySelector<HTMLButtonElement>("#up-check")?.addEventListener("click", async () => {
    const msg = vSection.querySelector<HTMLElement>("#up-msg")!;
    const info = await api.CheckForUpdate();
    msg.textContent = info.available ? `Vinyl ${info.version} est disponible (voir le bandeau).` : "Vinyl est à jour.";
  });
  vSection.querySelector<HTMLButtonElement>("#beta-on")?.addEventListener("click", async () => {
    const code = vSection.querySelector<HTMLInputElement>("#beta-code")!.value;
    try { await api.SetBetaCode(code); renderVersion(); }
    catch { vSection.querySelector<HTMLElement>("#beta-msg")!.textContent = "Code invalide"; }
  });
  vSection.querySelector<HTMLButtonElement>("#beta-off")?.addEventListener("click", async () => { await api.SetBetaCode(""); renderVersion(); });
};
renderVersion();
```
et dans le gabarit de la page, une `<section id="version-section" class="flex flex-col gap-3 border-t py-5">` (même style que `#spotify-section`).

- [ ] **Step 4: Run tests**

Run: `cd frontend && npx tsc --noEmit && npm test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend
git commit -m "feat(mises à jour): bandeau de mise à jour et section Version / Contributeur"
```

---

### Task 7: Release (fichiers de mise à jour, empreintes, version)

**Files:**
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: Update the workflow**

Job `mac` :
```yaml
      - name: Version
        run: |
          V="${GITHUB_REF_NAME#v}"
          echo "VINYL_VERSION=$V" >> "$GITHUB_ENV"
          /usr/libexec/PlistBuddy -c "Set :CFBundleShortVersionString $V" -c "Set :CFBundleVersion $V" build/darwin/Info.plist
      - run: CGO_ENABLED=1 wails3 package
      - run: ditto -c -k --keepParent bin/vinyl.app Vinyl-darwin-arm64.zip
      - uses: actions/upload-artifact@v4
        with: { name: mac, path: Vinyl-darwin-arm64.zip }
```
Job `windows` :
```yaml
      - name: Version
        shell: pwsh
        run: |
          $v = "$env:GITHUB_REF_NAME".TrimStart("v")
          Add-Content $env:GITHUB_ENV "VINYL_VERSION=$v"
          $j = Get-Content build/windows/info.json -Raw | ConvertFrom-Json
          $j.fixed.file_version = $v; $j.info."0000".ProductVersion = $v
          $j | ConvertTo-Json -Depth 5 | Set-Content build/windows/info.json
      - run: wails3 package
        env: { CGO_ENABLED: "0" }
      - name: Fichiers Windows
        shell: pwsh
        run: |
          Get-ChildItem bin -Filter *installer*.exe | Select-Object -First 1 | Copy-Item -Destination Vinyl-Setup.exe
          Compress-Archive -Path bin/vinyl.exe -DestinationPath Vinyl-windows-amd64.zip
      - uses: actions/upload-artifact@v4
        with: { name: windows, path: "Vinyl-Setup.exe\nVinyl-windows-amd64.zip" }
```
(écrire `path` en liste YAML multiligne :
```yaml
        with:
          name: windows
          path: |
            Vinyl-Setup.exe
            Vinyl-windows-amd64.zip
```
)
Job `release` : avant `softprops/action-gh-release`, une étape
```yaml
      - name: Empreintes
        run: |
          cp mac/* windows/* .
          sha256sum Vinyl-darwin-arm64.zip Vinyl-windows-amd64.zip Vinyl-Setup.exe > SHA256SUMS
```
et `files:` devient :
```yaml
          files: |
            Vinyl-darwin-arm64.zip
            Vinyl-windows-amd64.zip
            Vinyl-Setup.exe
            SHA256SUMS
```
Mettre à jour le texte `body` : Mac → `Vinyl-darwin-arm64.zip`.

- [ ] **Step 2: Validate**

Run: `ruby -ryaml -e 'YAML.load_file(".github/workflows/release.yml"); puts "ok"' && grep -c "Vinyl-darwin-arm64.zip\|Vinyl-windows-amd64.zip\|SHA256SUMS" .github/workflows/release.yml`
Expected: `ok` puis au moins `6`.

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/release.yml
git commit -m "ci(release): fichiers de mise à jour, empreintes SHA256SUMS et numéro de version"
```

---

### Task 8: Vérification réelle

**Files:**
- Modify: `docs/windows-verification.md` (section « Mises à jour »)

- [ ] **Step 1: Add the checklist**

```markdown
## Mises à jour (Mac et Windows)

1. Installer la release `vA` (première version avec les mises à jour) à la main.
2. Publier `vB` (`./release.sh`) ; relancer Vinyl : le bandeau « Vinyl B est disponible » apparaît.
3. « Mettre à jour » : téléchargement, puis Vinyl redémarre ; Réglages affiche « Vinyl B ».
4. Publier une bêta `vC-beta.1` : sans code, rien n'est proposé ; avec un code (`./beta-code.sh "Prénom"`)
   collé dans Réglages > Contributeur, la bêta est proposée ; « Retirer » : elle ne l'est plus.
5. Couper le réseau et relancer : aucun message, l'app démarre normalement.
```

- [ ] **Step 2: Commit**

```bash
git add docs/windows-verification.md
git commit -m "docs: vérification des mises à jour"
```

- [ ] **Step 3: Hand off**

Avec l'accord de l'utilisateur : créer la clé (`./beta-code.sh "Pierre"` → commiter `internal/beta/pubkey.go`), publier une version `vA` puis `vB` et dérouler la liste sur Mac et sur le PC Windows.
