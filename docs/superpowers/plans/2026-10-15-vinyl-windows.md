# Vinyl sur Windows — plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Vinyl fonctionne sur macOS et Windows ; une release GitHub fournit `Vinyl-mac.zip` et `Vinyl-Setup.exe`.

**Architecture:** Un seul code source ; l'assemblage propre à chaque système est dans `platform_darwin.go` / `platform_windows.go`. Sous Windows, un nouveau paquet `internal/winmedia` pilote Spotify et Apple Music par les contrôles multimédia de Windows (GSMTC) appelés depuis Go via `github.com/saltosystems/winrt-go` ; sa logique est testée partout avec des sessions factices, seule la couche WinRT est propre à Windows.

**Tech Stack:** Go 1.26, Wails v3.0.0-beta.29, TypeScript + Vitest, `github.com/saltosystems/winrt-go` (+ `github.com/go-ole/go-ole`), GitHub Actions, NSIS.

**Spec:** `docs/superpowers/specs/2026-10-15-vinyl-windows-design.md`

## Global Constraints

- Branche de travail : `feat/windows`, créée depuis `feat/refonte-graphique`.
- Tests Go sur Mac : `CGO_ENABLED=1 go test -race . ./internal/...` et `CGO_ENABLED=1 go vet . ./internal/...` (jamais `./...` : `build/ios`, `build/android`).
- Vérification Windows depuis le Mac (à chaque tâche Go) : `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/vinyl.exe .` puis `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./internal/...`.
- Frontend : `cd frontend && npx tsc --noEmit && npm test`.
- Liaisons Wails après tout changement de méthode exportée du service : `PATH="$PATH:$(go env GOPATH)/bin" wails3 generate bindings -clean=true -ts -i`.
- La version Mac ne perd aucune fonction ; aucun changement de comportement sous macOS hors ce que dit une tâche.
- Vinyl ne crée jamais rien dans la bibliothèque de l'utilisateur (test existant `TestNoScriptCreatesAPlaylist`).
- Messages de commit `type(portée): description` en français, terminés par `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.
- Textes de l'interface en français.
- Pas de version Mac Intel (arm64 seulement).

## Review Focus

1. Spotify installé depuis le Microsoft Store a un identifiant de session différent (`SpotifyAB.SpotifyMusic_…!Spotify`) : il doit être reconnu comme « spotify » (test Task 3).
2. Une session sans titre (publicité, changement de morceau) ne doit pas entrer dans la pile ni le compteur : `TrackID` vide, état « stopped » (test Task 3).
3. Windows relève la position seulement de temps en temps : pendant la lecture, la position doit avancer avec le temps écoulé depuis ce relevé, sans dépasser la durée (test Task 3).
4. Une pochette illisible (miniature absente, erreur WinRT) ne doit ni planter ni être relue à chaque sondage (test Task 6).
5. L'API web Spotify refuse la lecture (403 sans Premium, 404 sans appareil actif) : repli sur l'ouverture du lien `spotify:` au lieu d'une erreur muette (test Task 8).

---

### Task 1: Assemblage par plateforme

**Files:**
- Create: `platform_darwin.go`, `platform_windows.go`
- Modify: `main.go:53-56,66-67,117-129,213` (lecteurs, bibliothèque Apple Music, widget), `internal/app/service.go:36-39` (`New` sans bibliothèque), `internal/app/service.go:166-175` (`ItemArtwork` sans bibliothèque)
- Test: `internal/app/platform_test.go`

**Interfaces:**
- Produces: `type platform struct { players []player.Player; music *catalog.MusicLibrary }`, `func newPlatform(artworkDir string) platform`, `func newWidgetManager(cfg settings.Settings, cacheDir, settingsPath string, emit func(widgetupdate.Status)) *widgetupdate.Manager` (nil hors Mac). `app.New(store, watcher, nil)` accepté : pas de catalogue « applemusic ».

- [ ] **Step 1: Write the failing test** — `internal/app/platform_test.go`

```go
package app

import (
	"errors"
	"testing"
	"time"

	"vinyl/internal/catalog"
	"vinyl/internal/player"
)

// Sous Windows, pas de bibliothèque Apple Music : le service doit le dire, pas planter.
func TestServiceWithoutAppleMusicLibrary(t *testing.T) {
	w := player.NewWatcher(nil, func() string { return "auto" }, func(player.Snapshot) {}, time.Now)
	s := New(nil, w, nil)
	if _, err := s.Library("applemusic", "albums", 0); !errors.Is(err, catalog.ErrUnsupported) {
		t.Errorf("Library : %v", err)
	}
	if u, err := s.ItemArtwork("applemusic", "album:X"); u != "" || err != nil {
		t.Errorf("ItemArtwork : %q %v", u, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `CGO_ENABLED=1 go test ./internal/app/ -run TestServiceWithoutAppleMusicLibrary`
Expected: FAIL (panic : appel sur un `*MusicLibrary` nil).

- [ ] **Step 3: Implement** — dans `internal/app/service.go`, `New` :

```go
func New(store *settings.Store, watcher *player.Watcher, music *catalog.MusicLibrary) *VinylService {
	s := &VinylService{store: store, watcher: watcher, music: music, catalogs: map[string]catalog.Catalog{}}
	if music != nil { // bibliothèque Apple Music : Mac seulement
		s.catalogs["applemusic"] = music
	}
	return s
}
```

et au début de `ItemArtwork` : `if source != "applemusic" || s.music == nil { return "", nil }`.

`platform_darwin.go` :

```go
//go:build darwin

package main

import (
	"path/filepath"
	"time"

	"vinyl/internal/apps"
	"vinyl/internal/catalog"
	"vinyl/internal/player"
	"vinyl/internal/settings"
	"vinyl/internal/widgetupdate"
)

// platform : pièces propres au système. Mac : Musique et Spotify par AppleScript, bibliothèque Apple Music.
type platform struct {
	players []player.Player
	music   *catalog.MusicLibrary
}

func newPlatform(artworkDir string) platform {
	player.IsRunning = apps.Running // une app non lancée n'est pas interrogée (pas d'osascript)
	runner := player.OSAScript{}
	return platform{
		players: []player.Player{player.NewAppleMusic(runner, artworkDir), player.NewSpotify(runner)},
		music:   catalog.NewMusicLibrary(runner, catalog.NewITunes(), artworkDir),
	}
}

// newWidgetManager : mise à jour du widget de bureau (WidgetKit), Mac seulement.
func newWidgetManager(cfg settings.Settings, cacheDir, settingsPath string, emit func(widgetupdate.Status)) *widgetupdate.Manager {
	widgetDir := cfg.WidgetProjectDir
	if widgetDir == "" {
		widgetDir = projectDir
	}
	return widgetupdate.NewManager(&widgetupdate.Updater{
		ProjectDir:    widgetDir,
		AppPath:       "/Applications/Vinyl Widget.app",
		BuildDir:      filepath.Join(cacheDir, "Vinyl", "widget-build"),
		RecordPath:    filepath.Join(filepath.Dir(settingsPath), "widget-install.json"),
		XcodeGenPaths: []string{"/opt/homebrew/bin/xcodegen", "/usr/local/bin/xcodegen"},
		Run:           widgetupdate.ExecRunner{},
		Now:           time.Now,
	}, emit)
}
```

`platform_windows.go` (les lecteurs arrivent en Task 7) :

```go
//go:build windows

package main

import (
	"vinyl/internal/catalog"
	"vinyl/internal/player"
	"vinyl/internal/settings"
	"vinyl/internal/widgetupdate"
)

// platform : pièces propres au système. Windows : contrôles multimédia (Task 7), pas de bibliothèque Apple Music.
type platform struct {
	players []player.Player
	music   *catalog.MusicLibrary
}

func newPlatform(artworkDir string) platform { return platform{} }

// newWidgetManager : pas de widget de bureau sous Windows.
func newWidgetManager(settings.Settings, string, string, func(widgetupdate.Status)) *widgetupdate.Manager {
	return nil
}
```

Dans `main.go` : remplacer les lignes `player.IsRunning = …`, `runner := …`, `players := …` par `plat := newPlatform(artworkDir)` et `players := plat.players` ; remplacer `music := catalog.NewMusicLibrary(…)` puis `app.New(store, watcher, music)` par `app.New(store, watcher, plat.music)` ; remplacer le bloc `widgetDir … service.SetWidgetUpdates(widgetMgr)` par :

```go
	widgetMgr := newWidgetManager(store.Get(), cacheDir, settingsPath, func(s widgetupdate.Status) { wailsApp.Event.Emit("widgetupdate", s) })
	if widgetMgr != nil {
		service.SetWidgetUpdates(widgetMgr)
	}
```

et `go widgetMgr.Run(ctx, 6*time.Hour)` par `if widgetMgr != nil { go widgetMgr.Run(ctx, 6*time.Hour) }`. Retirer les imports devenus inutiles (`apps`, `catalog` si plus utilisé hors `SpotifyWeb`, `widgetupdate` reste pour l'événement).

- [ ] **Step 4: Run tests and both builds**

Run: `CGO_ENABLED=1 go test -race . ./internal/... && CGO_ENABLED=1 go vet . ./internal/... && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/vinyl.exe . && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./internal/...`
Expected: tout passe ; `/tmp/vinyl.exe` produit.

- [ ] **Step 5: Commit**

```bash
git add platform_darwin.go platform_windows.go main.go internal/app/service.go internal/app/platform_test.go
git commit -m "refactor(plateforme): assemblage propre à chaque système (Mac, Windows)"
```

---

### Task 2: Plateforme côté service et interface

**Files:**
- Create: `internal/app/openurl_darwin.go`, `internal/app/openurl_windows.go`, `frontend/src/ui/platform.ts`
- Modify: `internal/app/service.go:132-135` (`OpenAutomationSettings`), `internal/app/service.go:194-202` (`OpenURL`), `frontend/src/main.ts:64-66` (bandeau du widget), `frontend/src/views/bibliotheque.ts:328-329` (message Apple Music), `frontend/src/views/shelf.ts`, `frontend/src/views/platine.ts:21` (raccourci mini-platine)
- Test: `internal/app/platform_test.go`, `frontend/src/views/shelf.test.ts`

**Interfaces:**
- Consumes: Task 1.
- Produces: `func (s *VinylService) Platform() string` (« darwin » | « windows »), frontend `platform(): Promise<string>` (`ui/platform.ts`), `unavailableLibraryMessage(platform: string): string` (`views/shelf.ts`).

- [ ] **Step 1: Write the failing tests**

`internal/app/platform_test.go` (ajout) :

```go
func TestPlatform(t *testing.T) {
	if p := New(nil, player.NewWatcher(nil, func() string { return "auto" }, func(player.Snapshot) {}, time.Now), nil).Platform(); p != runtime.GOOS {
		t.Errorf("Platform = %q", p)
	}
}
```
(ajouter `"runtime"` aux imports).

`frontend/src/views/shelf.test.ts` (ajout) :

```ts
test("Apple Music sans bibliothèque : message selon la plateforme", () => {
  expect(unavailableLibraryMessage("windows")).toBe("La bibliothèque Apple Music n'est disponible que sur Mac.");
  expect(unavailableLibraryMessage("darwin")).toBe("Non disponible pour Apple Music.");
});
```
(ajouter `unavailableLibraryMessage` à l'import de `./shelf`).

- [ ] **Step 2: Run tests to verify they fail**

Run: `CGO_ENABLED=1 go test ./internal/app/ -run TestPlatform` → FAIL (`Platform` undefined) ; `cd frontend && npx vitest run src/views/shelf.test.ts` → FAIL (`unavailableLibraryMessage` absent).

- [ ] **Step 3: Implement**

`service.go` :

```go
// Platform : « darwin » ou « windows » ; le frontend masque ce qui n'existe pas sur le système.
func (s *VinylService) Platform() string { return runtime.GOOS }
```

`OpenAutomationSettings` : en tête, `if runtime.GOOS != "darwin" { return nil } // réglage propre au Mac`.

`OpenURL` : garder la validation des préfixes, remplacer `return exec.Command("open", u).Run()` et la conversion `music://` par `return openURL(u)`.

`internal/app/openurl_darwin.go` :

```go
//go:build darwin

package app

import (
	"os/exec"
	"strings"
)

// openURL : les liens Apple Music s'ouvrent dans l'app Musique.
func openURL(u string) error {
	if strings.HasPrefix(u, "https://music.apple.com/") {
		u = "music://" + strings.TrimPrefix(u, "https://")
	}
	return exec.Command("open", u).Run()
}
```

`internal/app/openurl_windows.go` :

```go
//go:build windows

package app

import "os/exec"

// openURL : gestionnaire de protocoles de Windows (https:, spotify:).
func openURL(u string) error {
	return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Run()
}
```

Régénérer les liaisons Wails (Global Constraints).

`frontend/src/ui/platform.ts` :

```ts
// Système de l'app (« darwin » | « windows »), lu une fois : sert à masquer ce qui n'existe pas.
import { api } from "../api";

let cached: Promise<string> | null = null;
export const platform = (): Promise<string> => (cached ??= api.Platform().catch(() => "darwin"));
```

`frontend/src/views/shelf.ts` (ajout) :

```ts
/** Message quand la bibliothèque Apple Music n'est pas accessible (Windows : jamais). */
export function unavailableLibraryMessage(platform: string): string {
  return platform === "windows" ? "La bibliothèque Apple Music n'est disponible que sur Mac." : "Non disponible pour Apple Music.";
}
```

`bibliotheque.ts`, branche `msg.includes("non disponible")` : `platform().then((p) => (status.textContent = unavailableLibraryMessage(p)));`.

`main.ts` : n'afficher le bandeau de mise à jour du widget que sur Mac — entourer `api.WidgetUpdateStatus().then(showWidgetUpdate)` et l'abonnement `onWidgetUpdate(showWidgetUpdate)` de `platform().then((p) => { if (p !== "darwin") return; … })`.

`platine.ts` : le titre du bouton mini-platine devient `platform().then((p) => ($("pl-mini").title = p === "darwin" ? "Mini-platine (⌘⇧M)" : "Mini-platine (Ctrl+Maj+M)"))`.

- [ ] **Step 4: Run tests and builds**

Run: `CGO_ENABLED=1 go test -race . ./internal/... && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/vinyl.exe . && cd frontend && npx tsc --noEmit && npm test`
Expected: tout passe.

- [ ] **Step 5: Commit**

```bash
git add internal/app frontend/src frontend/bindings
git commit -m "feat(plateforme): le frontend sait sur quel système il tourne"
```

---

### Task 3: Lecteur Windows — logique (`internal/winmedia`)

**Files:**
- Create: `internal/winmedia/session.go`, `internal/winmedia/player.go`
- Test: `internal/winmedia/player_test.go`

**Interfaces:**
- Produces:
  - `type Session interface { AppID() string; Info() (Info, error); Thumbnail() ([]byte, error); TogglePlayPause() error; Next() error; Previous() error; Seek(time.Duration) error; SetShuffle(bool) error; SetRepeat(player.RepeatMode) error }`
  - `type Info struct { Title, Artist, Album string; Position, Duration time.Duration; Status player.State; Shuffle bool; Repeat player.RepeatMode; UpdatedAt time.Time }`
  - `type Sessions interface { List() ([]Session, error) }`
  - `type Source struct { Name string; Match func(appID string) bool }`, `var SpotifySource, AppleMusicSource Source`
  - `func TrackID(title, artist, album string) string`
  - `type Player struct`, `func NewPlayer(src Source, s Sessions, now func() time.Time) *Player` ; implémente `player.Player` et `player.Modes`.

- [ ] **Step 1: Write the failing test** — `internal/winmedia/player_test.go`

```go
package winmedia

import (
	"context"
	"errors"
	"testing"
	"time"

	"vinyl/internal/player"
)

type fakeSession struct {
	id        string
	info      Info
	calls     []string
	seeks     []time.Duration
	thumb     []byte
	thumbErr  error
	thumbRead int
}

func (f *fakeSession) AppID() string          { return f.id }
func (f *fakeSession) Info() (Info, error)    { return f.info, nil }
func (f *fakeSession) Thumbnail() ([]byte, error) {
	f.thumbRead++
	return f.thumb, f.thumbErr
}
func (f *fakeSession) TogglePlayPause() error { f.calls = append(f.calls, "playpause"); return nil }
func (f *fakeSession) Next() error            { f.calls = append(f.calls, "next"); return nil }
func (f *fakeSession) Previous() error        { f.calls = append(f.calls, "previous"); return nil }
func (f *fakeSession) Seek(d time.Duration) error {
	f.seeks = append(f.seeks, d)
	return nil
}
func (f *fakeSession) SetShuffle(on bool) error {
	f.info.Shuffle = on
	return nil
}
func (f *fakeSession) SetRepeat(m player.RepeatMode) error {
	f.info.Repeat = m
	return nil
}

type fakeSessions []Session

func (f fakeSessions) List() ([]Session, error) { return f, nil }

var t0 = time.Unix(1_000_000, 0)

func song(title string) Info {
	return Info{Title: title, Artist: "A", Album: "B", Position: 10 * time.Second, Duration: 200 * time.Second,
		Status: player.StatePlaying, UpdatedAt: t0}
}

func TestNotRunningWithoutSession(t *testing.T) {
	p := NewPlayer(SpotifySource, fakeSessions{&fakeSession{id: "Chrome", info: song("x")}}, func() time.Time { return t0 })
	if _, err := p.NowPlaying(context.Background()); !errors.Is(err, player.ErrNotRunning) {
		t.Errorf("err = %v", err)
	}
}

// Spotify classique et Spotify du Microsoft Store ; Apple Music pour Windows.
func TestSourcesAreRecognised(t *testing.T) {
	for id, src := range map[string]Source{
		"Spotify.exe":                            SpotifySource,
		"SpotifyAB.SpotifyMusic_zpdnekdrzrea0!Spotify": SpotifySource,
		"AppleInc.AppleMusicWin_nzyj5cx40ttqa!App": AppleMusicSource,
	} {
		if !src.Match(id) {
			t.Errorf("%s non reconnu comme %s", id, src.Name)
		}
	}
	if SpotifySource.Match("AppleInc.AppleMusicWin_x!App") || AppleMusicSource.Match("Spotify.exe") {
		t.Error("confusion entre les sources")
	}
}

func TestNowPlaying(t *testing.T) {
	s := &fakeSession{id: "Spotify.exe", info: song("Creep")}
	p := NewPlayer(SpotifySource, fakeSessions{s}, func() time.Time { return t0.Add(3 * time.Second) })
	np, err := p.NowPlaying(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if np.Source != "spotify" || np.Title != "Creep" || np.State != player.StatePlaying || np.DurationMs != 200000 {
		t.Errorf("np = %+v", np)
	}
	if np.PositionMs != 13000 { // relevé à 10 s, 3 s plus tôt
		t.Errorf("position = %d", np.PositionMs)
	}
	if np.TrackID == "" || np.TrackID != TrackID("Creep", "A", "B") || TrackID("Creep", "A", "B") == TrackID("Creep", "A", "C") {
		t.Errorf("TrackID = %q", np.TrackID)
	}
}

func TestPositionNeverExceedsDuration(t *testing.T) {
	s := &fakeSession{id: "Spotify.exe", info: song("Creep")}
	p := NewPlayer(SpotifySource, fakeSessions{s}, func() time.Time { return t0.Add(time.Hour) })
	if np, _ := p.NowPlaying(context.Background()); np.PositionMs != 200000 {
		t.Errorf("position = %d", np.PositionMs)
	}
	s.info.Status = player.StatePaused // en pause : pas d'interpolation
	if np, _ := p.NowPlaying(context.Background()); np.PositionMs != 10000 {
		t.Errorf("position en pause = %d", np.PositionMs)
	}
}

// Publicité, changement de morceau : pas de titre → ni pile ni compteur.
func TestUntitledSessionIsStopped(t *testing.T) {
	s := &fakeSession{id: "Spotify.exe", info: Info{Status: player.StatePlaying}}
	np, err := NewPlayer(SpotifySource, fakeSessions{s}, time.Now).NowPlaying(context.Background())
	if err != nil || np.TrackID != "" || np.State != player.StateStopped {
		t.Errorf("np = %+v err = %v", np, err)
	}
}

func TestCommandsAndModes(t *testing.T) {
	s := &fakeSession{id: "AppleInc.AppleMusicWin_x!App", info: song("x")}
	p := NewPlayer(AppleMusicSource, fakeSessions{s}, time.Now)
	ctx := context.Background()
	p.PlayPause(ctx)
	p.Next(ctx)
	p.Previous(ctx)
	p.Seek(ctx, 42*time.Second)
	if len(s.calls) != 3 || s.calls[2] != "previous" || len(s.seeks) != 1 || s.seeks[0] != 42*time.Second {
		t.Errorf("calls %v seeks %v", s.calls, s.seeks)
	}
	p.SetShuffle(ctx, true)
	p.SetRepeat(ctx, player.RepeatAll)
	if sh, rep, err := p.PlaybackModes(ctx); !sh || rep != player.RepeatAll || err != nil {
		t.Errorf("modes %v %v %v", sh, rep, err)
	}
	if err := p.PlayItem(ctx, "album:x"); err == nil {
		t.Error("PlayItem : non disponible sous Windows")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/winmedia/`
Expected: FAIL (paquet sans code).

- [ ] **Step 3: Implement**

`internal/winmedia/session.go` :

```go
// Package winmedia pilote les lecteurs sous Windows par les contrôles multimédia du système (GSMTC).
// La logique (player.go) est testée partout ; seule la couche WinRT (sessions_windows.go) est propre
// à Windows.
package winmedia

import (
	"time"

	"vinyl/internal/player"
)

// Session : une app qui joue, vue au travers des contrôles multimédia de Windows.
type Session interface {
	AppID() string // SourceAppUserModelId, ex. « Spotify.exe »
	Info() (Info, error)
	Thumbnail() ([]byte, error) // miniature de la pochette ; nil si aucune
	TogglePlayPause() error
	Next() error
	Previous() error
	Seek(pos time.Duration) error
	SetShuffle(on bool) error
	SetRepeat(mode player.RepeatMode) error
}

// Info : état d'une session.
type Info struct {
	Title, Artist, Album string
	Position, Duration   time.Duration
	Status               player.State
	Shuffle              bool
	Repeat               player.RepeatMode
	UpdatedAt            time.Time // instant du dernier relevé de position par Windows
}

// Sessions liste les sessions ouvertes.
type Sessions interface {
	List() ([]Session, error)
}
```

`internal/winmedia/player.go` :

```go
package winmedia

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"vinyl/internal/player"
)

// Source : une app reconnue par l'identifiant de sa session, sous le nom de source de Vinyl.
type Source struct {
	Name  string
	Match func(appID string) bool
}

var (
	// Spotify classique (Spotify.exe) ou du Microsoft Store (SpotifyAB.SpotifyMusic_…).
	SpotifySource = Source{Name: "spotify", Match: func(id string) bool {
		l := strings.ToLower(id)
		return l == "spotify.exe" || strings.HasPrefix(l, "spotifyab.spotifymusic")
	}}
	// Apple Music pour Windows (AppleInc.AppleMusicWin_…).
	AppleMusicSource = Source{Name: "applemusic", Match: func(id string) bool {
		return strings.HasPrefix(strings.ToLower(id), "appleinc.applemusic")
	}}
)

// TrackID : Windows ne donne pas d'identifiant de morceau ; empreinte stable titre + artiste + album.
func TrackID(title, artist, album string) string {
	sum := sha1.Sum([]byte(title + "\x1f" + artist + "\x1f" + album))
	return "w" + hex.EncodeToString(sum[:8])
}

// Player : un lecteur de Vinyl (Spotify ou Apple Music) sous Windows.
type Player struct {
	src      Source
	sessions Sessions
	now      func() time.Time
}

func NewPlayer(src Source, s Sessions, now func() time.Time) *Player {
	return &Player{src: src, sessions: s, now: now}
}

func (p *Player) Name() string { return p.src.Name }

// session : la session de l'app ; aucune → app non lancée.
func (p *Player) session() (Session, error) {
	list, err := p.sessions.List()
	if err != nil {
		return nil, err
	}
	for _, s := range list {
		if p.src.Match(s.AppID()) {
			return s, nil
		}
	}
	return nil, player.ErrNotRunning
}

func (p *Player) NowPlaying(context.Context) (player.NowPlaying, error) {
	s, err := p.session()
	if err != nil {
		return player.NowPlaying{}, err
	}
	in, err := s.Info()
	if err != nil {
		return player.NowPlaying{}, err
	}
	if in.Title == "" { // publicité, changement de morceau
		return player.NowPlaying{Source: p.src.Name, State: player.StateStopped}, nil
	}
	pos := in.Position
	if in.Status == player.StatePlaying && !in.UpdatedAt.IsZero() {
		pos += p.now().Sub(in.UpdatedAt) // Windows ne relève la position que de temps en temps
	}
	if in.Duration > 0 && pos > in.Duration {
		pos = in.Duration
	}
	return player.NowPlaying{
		Source: p.src.Name, State: in.Status, TrackID: TrackID(in.Title, in.Artist, in.Album),
		Title: in.Title, Artist: in.Artist, Album: in.Album,
		DurationMs: in.Duration.Milliseconds(), PositionMs: pos.Milliseconds(),
	}, nil
}

func (p *Player) do(f func(Session) error) error {
	s, err := p.session()
	if err != nil {
		return err
	}
	return f(s)
}

func (p *Player) PlayPause(context.Context) error { return p.do(Session.TogglePlayPause) }
func (p *Player) Next(context.Context) error      { return p.do(Session.Next) }
func (p *Player) Previous(context.Context) error  { return p.do(Session.Previous) }
func (p *Player) Seek(_ context.Context, pos time.Duration) error {
	return p.do(func(s Session) error { return s.Seek(pos) })
}

// PlayItem : les contrôles de Windows ne savent pas lancer un album ; Spotify passe par l'API web (service).
func (p *Player) PlayItem(context.Context, string) error {
	return errors.New("non disponible sous Windows")
}

func (p *Player) PlaybackModes(context.Context) (bool, player.RepeatMode, error) {
	s, err := p.session()
	if err != nil {
		return false, player.RepeatOff, err
	}
	in, err := s.Info()
	return in.Shuffle, in.Repeat, err
}

func (p *Player) SetShuffle(_ context.Context, on bool) error {
	return p.do(func(s Session) error { return s.SetShuffle(on) })
}

func (p *Player) SetRepeat(_ context.Context, m player.RepeatMode) error {
	return p.do(func(s Session) error { return s.SetRepeat(m) })
}
```

- [ ] **Step 4: Run tests**

Run: `go test -race ./internal/winmedia/ && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./internal/winmedia/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/winmedia
git commit -m "feat(windows): lecteur par les contrôles multimédia de Windows (logique)"
```

---

### Task 4: Réécoute depuis la pile sous Windows

**Files:**
- Create: `internal/winmedia/replay.go`
- Modify: `internal/winmedia/player.go` (champ `sleep`)
- Test: `internal/winmedia/replay_test.go`

**Interfaces:**
- Consumes: Task 3 (`Player`, `Session`, `TrackID`).
- Produces: `func (p *Player) ReplayBack(ctx context.Context, trackID string, steps, posSec int) (int, error)`, `func (p *Player) JumpForward(ctx context.Context, steps int, trackID string, posSec int) error` — implémentent `player.BackReplayer`.

- [ ] **Step 1: Write the failing test** — `internal/winmedia/replay_test.go`

```go
package winmedia

import (
	"context"
	"testing"
	"time"

	"vinyl/internal/player"
)

// queueSession : une file de morceaux ; précédent / suivant déplacent la lecture.
type queueSession struct {
	fakeSession
	queue []string
	idx   int
}

func (q *queueSession) Info() (Info, error) { return song(q.queue[q.idx]), nil }
func (q *queueSession) Previous() error {
	if q.idx > 0 {
		q.idx--
	}
	return nil
}
func (q *queueSession) Next() error {
	if q.idx < len(q.queue)-1 {
		q.idx++
	}
	return nil
}

func replayPlayer(q *queueSession) *Player {
	p := NewPlayer(SpotifySource, fakeSessions{q}, time.Now)
	p.sleep = func(time.Duration) {}
	return p
}

func id(title string) string { return TrackID(title, "A", "B") }

func TestReplayBackFindsTheTrack(t *testing.T) {
	q := &queueSession{fakeSession: fakeSession{id: "Spotify.exe"}, queue: []string{"1", "2", "3", "4"}, idx: 3}
	taken, err := replayPlayer(q).ReplayBack(context.Background(), id("2"), 2, 61)
	if err != nil || taken != 2 || q.idx != 1 {
		t.Errorf("taken=%d idx=%d err=%v", taken, q.idx, err)
	}
}

// Pile décalée : le morceau est un cran plus loin.
func TestReplayBackOneStepFurther(t *testing.T) {
	q := &queueSession{fakeSession: fakeSession{id: "Spotify.exe"}, queue: []string{"1", "2", "3", "4"}, idx: 3}
	if taken, _ := replayPlayer(q).ReplayBack(context.Background(), id("1"), 2, 61); taken != 3 || q.idx != 0 {
		t.Errorf("taken=%d idx=%d", taken, q.idx)
	}
}

// Introuvable : retour au morceau de départ, à sa position.
func TestReplayBackNotFoundGoesBack(t *testing.T) {
	q := &queueSession{fakeSession: fakeSession{id: "Spotify.exe"}, queue: []string{"1", "2", "3", "4"}, idx: 3}
	taken, err := replayPlayer(q).ReplayBack(context.Background(), id("zz"), 1, 61)
	if err != nil || taken != 0 || q.idx != 3 {
		t.Fatalf("taken=%d idx=%d err=%v", taken, q.idx, err)
	}
	if len(q.seeks) != 1 || q.seeks[0] != 61*time.Second {
		t.Errorf("position non rétablie : %v", q.seeks)
	}
}

func TestJumpForward(t *testing.T) {
	q := &queueSession{fakeSession: fakeSession{id: "Spotify.exe"}, queue: []string{"1", "2", "3", "4"}, idx: 1}
	if err := replayPlayer(q).JumpForward(context.Background(), 2, id("4"), 30); err != nil {
		t.Fatal(err)
	}
	if q.idx != 3 || len(q.seeks) != 1 || q.seeks[0] != 30*time.Second {
		t.Errorf("idx=%d seeks=%v", q.idx, q.seeks)
	}
}

var _ player.BackReplayer = (*Player)(nil)
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/winmedia/`
Expected: FAIL (`p.sleep`, `ReplayBack`, `JumpForward` absents).

- [ ] **Step 3: Implement**

Dans `player.go`, ajouter le champ `sleep func(time.Duration)` à `Player` et l'initialiser à `time.Sleep` dans `NewPlayer`.

`internal/winmedia/replay.go` :

```go
package winmedia

import (
	"context"
	"time"
)

// current : identifiant du morceau en cours ("" si aucun).
func (p *Player) current() string {
	s, err := p.session()
	if err != nil {
		return ""
	}
	in, err := s.Info()
	if err != nil || in.Title == "" {
		return ""
	}
	return TrackID(in.Title, in.Artist, in.Album)
}

// waitFor attend (≤ 1,5 s) que le morceau visé soit celui en cours (fondus, changements lents).
func (p *Player) waitFor(trackID string) bool {
	for range 15 {
		if p.current() == trackID {
			return true
		}
		p.sleep(100 * time.Millisecond)
	}
	return p.current() == trackID
}

// ReplayBack réécoute un morceau sans toucher à la file du lecteur : « précédent » steps fois, puis un
// ou deux crans de plus si la pile est décalée. Introuvable : retour par « suivant » au morceau de
// départ, à sa position ; renvoie 0.
func (p *Player) ReplayBack(_ context.Context, trackID string, steps, posSec int) (int, error) {
	s, err := p.session()
	if err != nil {
		return 0, err
	}
	taken := 0
	for range steps {
		if err := s.Previous(); err != nil {
			return 0, err
		}
		taken++
		p.sleep(100 * time.Millisecond)
	}
	for extra := 0; extra <= 2; extra++ {
		if extra > 0 {
			if err := s.Previous(); err != nil {
				return 0, err
			}
			taken++
		}
		if p.waitFor(trackID) {
			return taken, nil
		}
	}
	for range taken {
		s.Next()
		p.sleep(100 * time.Millisecond)
	}
	p.sleep(300 * time.Millisecond)
	s.Seek(time.Duration(posSec) * time.Second)
	return 0, nil
}

// JumpForward revient au morceau interrompu par « suivant » et le replace où il en était.
func (p *Player) JumpForward(_ context.Context, steps int, trackID string, posSec int) error {
	s, err := p.session()
	if err != nil {
		return err
	}
	for range steps {
		if err := s.Next(); err != nil {
			return err
		}
		p.sleep(100 * time.Millisecond)
	}
	if p.waitFor(trackID) {
		return s.Seek(time.Duration(posSec) * time.Second)
	}
	return nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test -race ./internal/winmedia/ && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./internal/winmedia/`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/winmedia
git commit -m "feat(windows): réécoute depuis la pile par précédent / suivant"
```

---

### Task 5: Sessions réelles (WinRT, Windows seulement)

**Files:**
- Create: `internal/winmedia/sessions_windows.go`, `internal/winmedia/winrt_windows.go`
- Modify: `go.mod`, `go.sum` (`github.com/saltosystems/winrt-go`, `github.com/go-ole/go-ole`)

**Interfaces:**
- Consumes: Task 3 (`Session`, `Sessions`, `Info`).
- Produces: `func NewSystemSessions() (Sessions, error)` (Windows seulement). Toutes les appels WinRT passent par un fil système dédié (`winrt_windows.go`) : `func call(f func() error) error`.

Cette couche n'est vérifiable que sur Windows : ici, compilation croisée et `go vet` ; le comportement est vérifié par la liste manuelle de la Task 10.

- [ ] **Step 1: Add the dependency**

Run: `go get github.com/saltosystems/winrt-go@latest github.com/go-ole/go-ole@v1.3.0`
Expected: `go.mod` mis à jour.

- [ ] **Step 2: Implement the WinRT thread** — `internal/winmedia/winrt_windows.go`

```go
//go:build windows

package winmedia

import (
	"errors"
	"runtime"
	"time"
	"unsafe"

	"github.com/go-ole/go-ole"
	"github.com/saltosystems/winrt-go/windows/foundation"
)

// Tous les appels WinRT passent par un fil système unique, initialisé une fois (RoInitialize).
var jobs = make(chan func(), 16)

func init() {
	go func() {
		runtime.LockOSThread()
		if err := ole.RoInitialize(1); err != nil { // 1 = multithread (MTA)
			panic(err)
		}
		for f := range jobs {
			f()
		}
	}()
}

// call exécute f sur le fil WinRT et attend son résultat.
func call(f func() error) error {
	done := make(chan error, 1)
	jobs <- func() { done <- f() }
	return <-done
}

// await attend la fin d'une opération asynchrone WinRT (≤ 3 s) et renvoie son résultat brut.
func await(op *foundation.IAsyncOperation) (unsafe.Pointer, error) {
	if op == nil {
		return nil, errors.New("winrt: opération nulle")
	}
	defer op.Release()
	itf, err := op.QueryInterface(ole.NewGUID(foundation.GUIDIAsyncInfo))
	if err != nil {
		return nil, err
	}
	info := (*foundation.IAsyncInfo)(unsafe.Pointer(itf))
	defer info.Release()
	for deadline := time.Now().Add(3 * time.Second); time.Now().Before(deadline); time.Sleep(5 * time.Millisecond) {
		st, err := info.GetStatus()
		if err != nil {
			return nil, err
		}
		switch st {
		case foundation.AsyncStatusCompleted:
			return op.GetResults()
		case foundation.AsyncStatusError, foundation.AsyncStatusCanceled:
			return nil, errors.New("winrt: opération échouée")
		}
	}
	return nil, errors.New("winrt: délai dépassé")
}
```

(Vérifier les noms exacts des constantes de `foundation.AsyncStatus` dans `asyncstatus.go` du module et les reprendre tels quels.)

- [ ] **Step 3: Implement the sessions** — `internal/winmedia/sessions_windows.go`

```go
//go:build windows

package winmedia

import (
	"time"

	"github.com/saltosystems/winrt-go/windows/foundation"
	"github.com/saltosystems/winrt-go/windows/media"
	"github.com/saltosystems/winrt-go/windows/media/control"

	"vinyl/internal/player"
)

type systemSessions struct {
	mgr *control.GlobalSystemMediaTransportControlsSessionManager
}

// NewSystemSessions ouvre le gestionnaire des contrôles multimédia de Windows.
func NewSystemSessions() (Sessions, error) {
	var s systemSessions
	err := call(func() error {
		op, err := control.GlobalSystemMediaTransportControlsSessionManagerRequestAsync()
		if err != nil {
			return err
		}
		res, err := await(op)
		if err != nil {
			return err
		}
		s.mgr = (*control.GlobalSystemMediaTransportControlsSessionManager)(res)
		return nil
	})
	return &s, err
}

func (s *systemSessions) List() ([]Session, error) {
	var out []Session
	err := call(func() error {
		vec, err := s.mgr.GetSessions()
		if err != nil {
			return err
		}
		defer vec.Release()
		n, err := vec.GetSize()
		if err != nil {
			return err
		}
		for i := uint32(0); i < n; i++ {
			p, err := vec.GetAt(i)
			if err != nil {
				return err
			}
			ses := (*control.GlobalSystemMediaTransportControlsSession)(p)
			id, err := ses.GetSourceAppUserModelId()
			if err != nil {
				continue
			}
			out = append(out, &systemSession{s: ses, id: id})
		}
		return nil
	})
	return out, err
}

type systemSession struct {
	s  *control.GlobalSystemMediaTransportControlsSession
	id string
}

func (x *systemSession) AppID() string { return x.id }

// ticks : unités de 100 ns de WinRT.
func ticks(v int64) time.Duration { return time.Duration(v) * 100 }

func (x *systemSession) Info() (Info, error) {
	var in Info
	err := call(func() error {
		op, err := x.s.TryGetMediaPropertiesAsync()
		if err != nil {
			return err
		}
		res, err := await(op)
		if err != nil {
			return err
		}
		props := (*control.GlobalSystemMediaTransportControlsSessionMediaProperties)(res)
		in.Title, _ = props.GetTitle()
		in.Artist, _ = props.GetArtist()
		in.Album, _ = props.GetAlbumTitle()
		if tl, err := x.s.GetTimelineProperties(); err == nil {
			pos, _ := tl.GetPosition()
			end, _ := tl.GetEndTime()
			start, _ := tl.GetStartTime()
			upd, _ := tl.GetLastUpdatedTime()
			in.Position, in.Duration = ticks(pos.Duration), ticks(end.Duration-start.Duration)
			// DateTime : unités de 100 ns depuis le 1er janvier 1601 (UTC).
			in.UpdatedAt = time.Unix(0, (upd.UniversalTime-116444736000000000)*100)
		}
		if pi, err := x.s.GetPlaybackInfo(); err == nil {
			st, _ := pi.GetPlaybackStatus()
			switch st {
			case control.GlobalSystemMediaTransportControlsSessionPlaybackStatusPlaying:
				in.Status = player.StatePlaying
			case control.GlobalSystemMediaTransportControlsSessionPlaybackStatusPaused:
				in.Status = player.StatePaused
			default:
				in.Status = player.StateStopped
			}
			if ref, err := pi.GetIsShuffleActive(); err == nil && ref != nil {
				v, _ := ref.GetValue()
				in.Shuffle = uintptr(v)&0xff != 0 // IReference<bool>
			}
			if ref, err := pi.GetAutoRepeatMode(); err == nil && ref != nil {
				v, _ := ref.GetValue()
				switch media.MediaPlaybackAutoRepeatMode(int32(uintptr(v))) { // IReference<enum>
				case media.MediaPlaybackAutoRepeatModeTrack:
					in.Repeat = player.RepeatOne
				case media.MediaPlaybackAutoRepeatModeList:
					in.Repeat = player.RepeatAll
				default:
					in.Repeat = player.RepeatOff
				}
			}
		}
		return nil
	})
	return in, err
}

// try exécute une commande asynchrone de la session et attend sa fin.
func (x *systemSession) try(op func() (*foundation.IAsyncOperation, error)) error {
	return call(func() error {
		o, err := op()
		if err != nil {
			return err
		}
		_, err = await(o)
		return err
	})
}

func (x *systemSession) TogglePlayPause() error { return x.try(x.s.TryTogglePlayPauseAsync) }
func (x *systemSession) Next() error            { return x.try(x.s.TrySkipNextAsync) }
func (x *systemSession) Previous() error        { return x.try(x.s.TrySkipPreviousAsync) }
func (x *systemSession) Seek(pos time.Duration) error {
	return x.try(func() (*foundation.IAsyncOperation, error) {
		return x.s.TryChangePlaybackPositionAsync(int64(pos / 100))
	})
}
func (x *systemSession) SetShuffle(on bool) error {
	return x.try(func() (*foundation.IAsyncOperation, error) { return x.s.TryChangeShuffleActiveAsync(on) })
}
func (x *systemSession) SetRepeat(m player.RepeatMode) error {
	mode := media.MediaPlaybackAutoRepeatModeNone
	switch m {
	case player.RepeatOne:
		mode = media.MediaPlaybackAutoRepeatModeTrack
	case player.RepeatAll:
		mode = media.MediaPlaybackAutoRepeatModeList
	}
	return x.try(func() (*foundation.IAsyncOperation, error) { return x.s.TryChangeAutoRepeatModeAsync(mode) })
}

// Thumbnail : Task 6.
func (x *systemSession) Thumbnail() ([]byte, error) { return nil, nil }
```

- [ ] **Step 4: Cross-compile and vet**

Run: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./internal/winmedia/ && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/vinyl.exe . && CGO_ENABLED=1 go test -race . ./internal/...`
Expected: compile et `vet` sans remarque ; tests Mac verts (les fichiers `_windows.go` ne sont pas compilés sur Mac).

- [ ] **Step 5: Commit**

```bash
git add go.mod go.sum internal/winmedia
git commit -m "feat(windows): sessions multimédia réelles par WinRT"
```

---

### Task 6: Pochettes sous Windows

**Files:**
- Create: `internal/winmedia/artwork.go`, `internal/winmedia/winrtgen/` (liaisons générées), `internal/winmedia/thumbnail_windows.go`
- Modify: `internal/winmedia/player.go` (`NewPlayer` reçoit le dossier des pochettes ; `NowPlaying` remplit `ArtworkURL`), `internal/winmedia/sessions_windows.go` (`Thumbnail`)
- Test: `internal/winmedia/artwork_test.go`

**Interfaces:**
- Consumes: Task 3 (`Session.Thumbnail`), Task 5 (`call`, `await`).
- Produces: `func NewPlayer(src Source, s Sessions, now func() time.Time, artworkDir string) *Player` (nouvelle signature ; artworkDir "" = pas de pochette). URL servie : `/artwork/` + `player.ArtworkFileName(trackID)`, comme sur Mac.

- [ ] **Step 1: Write the failing test** — `internal/winmedia/artwork_test.go`

```go
package winmedia

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"vinyl/internal/player"
)

func TestArtworkWrittenOnceAndServed(t *testing.T) {
	dir := t.TempDir()
	s := &fakeSession{id: "Spotify.exe", info: song("Creep"), thumb: []byte("jpeg")}
	p := NewPlayer(SpotifySource, fakeSessions{s}, time.Now, dir)
	for range 3 {
		np, _ := p.NowPlaying(context.Background())
		want := "/artwork/" + player.ArtworkFileName(np.TrackID)
		if np.ArtworkURL != want {
			t.Fatalf("ArtworkURL = %q", np.ArtworkURL)
		}
	}
	if s.thumbRead != 1 {
		t.Errorf("miniature lue %d fois (une seule attendue)", s.thumbRead)
	}
	b, err := os.ReadFile(filepath.Join(dir, player.ArtworkFileName(TrackID("Creep", "A", "B"))))
	if err != nil || string(b) != "jpeg" {
		t.Errorf("fichier : %q %v", b, err)
	}
}

// Miniature illisible : pas de pochette, pas de plantage, et pas de nouvel essai à chaque sondage.
func TestArtworkFailureIsRemembered(t *testing.T) {
	s := &fakeSession{id: "Spotify.exe", info: song("Creep"), thumbErr: errors.New("winrt")}
	p := NewPlayer(SpotifySource, fakeSessions{s}, time.Now, t.TempDir())
	for range 3 {
		if np, err := p.NowPlaying(context.Background()); err != nil || np.ArtworkURL != "" {
			t.Fatalf("np=%+v err=%v", np, err)
		}
	}
	if s.thumbRead != 1 {
		t.Errorf("miniature relue %d fois", s.thumbRead)
	}
}
```

Mettre à jour les appels existants de `NewPlayer` dans `player_test.go` et `replay_test.go` en ajoutant `""` comme dernier argument.

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/winmedia/`
Expected: FAIL (signature de `NewPlayer`).

- [ ] **Step 3: Implement the logic** — `internal/winmedia/artwork.go`

```go
package winmedia

import (
	"os"
	"path/filepath"
	"sync"

	"vinyl/internal/player"
)

// artwork : pochettes écrites une fois par morceau dans le dossier servi sous /artwork/ ; un échec est
// retenu pour ne pas relire la miniature à chaque sondage.
type artwork struct {
	dir  string
	mu   sync.Mutex
	done map[string]string // trackID → URL ("" : pas de pochette)
}

func (a *artwork) url(trackID string, s Session) string {
	if a.dir == "" {
		return ""
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if u, ok := a.done[trackID]; ok {
		return u
	}
	u := ""
	if b, err := s.Thumbnail(); err == nil && len(b) > 0 {
		file := player.ArtworkFileName(trackID)
		if os.MkdirAll(a.dir, 0o755) == nil && os.WriteFile(filepath.Join(a.dir, file), b, 0o644) == nil {
			u = "/artwork/" + file
		}
	}
	a.done[trackID] = u
	return u
}
```

Dans `player.go` : champ `art *artwork` ; `NewPlayer(src, s, now, artworkDir string)` crée `&artwork{dir: artworkDir, done: map[string]string{}}` ; dans `NowPlaying`, après le calcul de `TrackID`, `ArtworkURL: p.art.url(id, s)`.

- [ ] **Step 4: Run tests**

Run: `go test -race ./internal/winmedia/`
Expected: PASS.

- [ ] **Step 5: Generate the stream bindings**

Run, depuis la racine :
```bash
mkdir -p internal/winmedia/winrtgen && cd internal/winmedia/winrtgen
go run github.com/saltosystems/winrt-go/cmd/winrt-go-gen -class Windows.Storage.Streams.DataReader -method-filter CreateDataReader -method-filter LoadAsync -method-filter ReadBytes -method-filter '!*'
go run github.com/saltosystems/winrt-go/cmd/winrt-go-gen -class Windows.Storage.Streams.DataReaderLoadOperation
go run github.com/saltosystems/winrt-go/cmd/winrt-go-gen -class Windows.Storage.Streams.IInputStream
go run github.com/saltosystems/winrt-go/cmd/winrt-go-gen -class Windows.Storage.Streams.IRandomAccessStreamWithContentType
cd ../../.. && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./internal/winmedia/...
```
Expected : fichiers générés sous `internal/winmedia/winrtgen/windows/storage/streams/` ; `vet` sans erreur. Si `vet` signale un type manquant, générer cette classe de la même façon. Les fichiers générés gardent leur en-tête « DO NOT EDIT » ; corriger seulement leur chemin d'import interne si `go vet` le demande (module `vinyl/internal/winmedia/winrtgen/...`).

- [ ] **Step 6: Implement the thumbnail read** — `internal/winmedia/thumbnail_windows.go`

```go
//go:build windows

package winmedia

import (
	"unsafe"

	"github.com/go-ole/go-ole"

	"vinyl/internal/winmedia/winrtgen/windows/storage/streams"
)

// readThumbnail lit la miniature d'une session : OpenReadAsync, puis DataReader sur le flux.
func (x *systemSession) readThumbnail() ([]byte, error) {
	var out []byte
	err := call(func() error {
		op, err := x.s.TryGetMediaPropertiesAsync()
		if err != nil {
			return err
		}
		res, err := await(op)
		if err != nil {
			return err
		}
		props := (*control.GlobalSystemMediaTransportControlsSessionMediaProperties)(res)
		ref, err := props.GetThumbnail()
		if err != nil || ref == nil {
			return err
		}
		openOp, err := ref.OpenReadAsync()
		if err != nil {
			return err
		}
		stream, err := await(openOp)
		if err != nil {
			return err
		}
		unk := (*ole.IUnknown)(stream)
		defer unk.Release()
		in, err := unk.QueryInterface(ole.NewGUID(streams.GUIDIInputStream))
		if err != nil {
			return err
		}
		defer in.Release()
		r, err := streams.DataReaderCreateDataReader((*streams.IInputStream)(unsafe.Pointer(in)))
		if err != nil {
			return err
		}
		defer r.Release()
		const max = 4 << 20 // 4 Mo : largement assez pour une miniature
		load, err := r.LoadAsync(max)
		if err != nil {
			return err
		}
		n, err := awaitLoad(load)
		if err != nil {
			return err
		}
		out, err = r.ReadBytes(n)
		return err
	})
	return out, err
}
```

`awaitLoad` : même logique que `await` (statut `IAsyncInfo` puis `GetResults`) pour `*streams.DataReaderLoadOperation`, renvoyant le `uint32` lu (`uint32(uintptr(res))`). Dans `sessions_windows.go`, remplacer `Thumbnail` par `func (x *systemSession) Thumbnail() ([]byte, error) { return x.readThumbnail() }` et ajouter l'import `control` à `thumbnail_windows.go`.

- [ ] **Step 7: Cross-compile and test**

Run: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./internal/winmedia/... && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/vinyl.exe . && CGO_ENABLED=1 go test -race . ./internal/...`
Expected: tout passe.

- [ ] **Step 8: Commit**

```bash
git add internal/winmedia
git commit -m "feat(windows): pochettes lues depuis les contrôles multimédia"
```

---

### Task 7: Brancher les lecteurs Windows

**Files:**
- Modify: `platform_windows.go`

**Interfaces:**
- Consumes: Task 1 (`platform`), Task 3/6 (`winmedia.NewPlayer`, `SpotifySource`, `AppleMusicSource`), Task 5 (`winmedia.NewSystemSessions`).

- [ ] **Step 1: Implement**

```go
func newPlatform(artworkDir string) platform {
	sessions, err := winmedia.NewSystemSessions()
	if err != nil {
		log.Printf("contrôles multimédia de Windows indisponibles : %v", err)
		return platform{}
	}
	return platform{players: []player.Player{
		winmedia.NewPlayer(winmedia.AppleMusicSource, sessions, time.Now, artworkDir),
		winmedia.NewPlayer(winmedia.SpotifySource, sessions, time.Now, artworkDir),
	}}
}
```
(imports `log`, `time`, `vinyl/internal/winmedia`).

- [ ] **Step 2: Cross-compile and test**

Run: `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/vinyl.exe . && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./internal/... && CGO_ENABLED=1 go test -race . ./internal/...`
Expected: tout passe.

- [ ] **Step 3: Commit**

```bash
git add platform_windows.go
git commit -m "feat(windows): Spotify et Apple Music branchés sur les contrôles multimédia"
```

---

### Task 8: Lancer un album Spotify sous Windows (API web)

**Files:**
- Create: `internal/catalog/spotifyplay.go`
- Modify: `internal/auth/spotifyauth.go:21` (scope), `internal/app/service.go` (`PlayFrom`, `PlayItem` pour Spotify hors Mac)
- Test: `internal/catalog/spotifyplay_test.go`, `internal/app/queue_test.go`

**Interfaces:**
- Consumes: `catalog.SpotifyWeb` (existant), `openURL` (Task 2).
- Produces: `func (c *SpotifyWeb) Play(ctx context.Context, contextURI, trackURI string, shuffle bool) error` ; `var ErrPlaybackRefused = errors.New(...)` renvoyée pour 403/404.

- [ ] **Step 1: Write the failing tests** — `internal/catalog/spotifyplay_test.go`

```go
package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
)

func TestSpotifyPlayStartsTrackInContext(t *testing.T) {
	var calls []string
	var body map[string]any
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
		calls = append(calls, r.Method+" "+r.URL.Path+"?"+r.URL.RawQuery)
		if r.URL.Path == "/me/player/play" {
			json.NewDecoder(r.Body).Decode(&body)
		}
		w.WriteHeader(204)
	})
	if err := c.Play(context.Background(), "spotify:album:A", "spotify:track:T", true); err != nil {
		t.Fatal(err)
	}
	if len(calls) != 2 || calls[0] != "PUT /me/player/shuffle?state=true" || calls[1] != "PUT /me/player/play?" {
		t.Errorf("appels %v", calls)
	}
	if body["context_uri"] != "spotify:album:A" || body["offset"].(map[string]any)["uri"] != "spotify:track:T" {
		t.Errorf("corps %v", body)
	}
}

// Pas Premium (403) ou aucun appareil (404) : erreur reconnaissable pour le repli.
func TestSpotifyPlayRefused(t *testing.T) {
	for _, code := range []int{403, 404} {
		c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(code) })
		if err := c.Play(context.Background(), "spotify:album:A", "", false); !errors.Is(err, ErrPlaybackRefused) {
			t.Errorf("%d : %v", code, err)
		}
	}
}
```

`internal/app/queue_test.go` (ajout) — un lecteur Spotify qui n'est pas le lecteur AppleScript :

```go
// Sous Windows, lancer un album Spotify passe par l'API web ; refusé → ouverture du lien spotify:.
func TestSpotifyPlayFromFallsBackToOpeningTheLink(t *testing.T) {
	p := &fakeModesPlayer{}
	s := serviceWith(p)
	var opened []string
	s.openLink = func(u string) error { opened = append(opened, u); return nil }
	s.spotifyPlay = func(context.Context, string, string, bool) error { return catalog.ErrPlaybackRefused }
	if err := s.PlayFrom("spotify", "spotify:album:A", "spotify:track:T", false); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != "spotify:album:A" {
		t.Errorf("ouvert : %v", opened)
	}
}
```

(`fakeModesPlayer.Name()` renvoie « applemusic » : le test passe la source « spotify », qui n'a pas de lecteur `*player.Spotify` — exactement le cas Windows.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `CGO_ENABLED=1 go test ./internal/catalog/ ./internal/app/`
Expected: FAIL (`Play`, `ErrPlaybackRefused`, `openLink`, `spotifyPlay` absents).

- [ ] **Step 3: Implement** — `internal/catalog/spotifyplay.go`

```go
package catalog

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// ErrPlaybackRefused : Spotify refuse de piloter la lecture (compte non Premium, aucun appareil actif).
var ErrPlaybackRefused = errors.New("Spotify refuse de lancer la lecture (Premium et app ouverte nécessaires)")

func (c *SpotifyWeb) put(ctx context.Context, path string, q url.Values, body any) error {
	tok, err := c.Token(ctx)
	if err != nil {
		return err
	}
	u := c.BaseURL + path
	if len(q) > 0 {
		u += "?" + q.Encode()
	}
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, err := http.NewRequestWithContext(ctx, "PUT", u, &buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	resp.Body.Close()
	switch resp.StatusCode {
	case 200, 202, 204:
		return nil
	case 403, 404:
		return ErrPlaybackRefused
	case 401:
		return fmt.Errorf("session Spotify expirée, reconnecte-toi")
	}
	return fmt.Errorf("spotify: HTTP %d", resp.StatusCode)
}

// Play lance un album / une playlist (contextURI) à partir d'un titre (trackURI, vide = début).
func (c *SpotifyWeb) Play(ctx context.Context, contextURI, trackURI string, shuffle bool) error {
	if err := c.put(ctx, "/me/player/shuffle", url.Values{"state": {strconv.FormatBool(shuffle)}}, nil); err != nil {
		return err
	}
	body := map[string]any{"context_uri": contextURI}
	if trackURI != "" {
		body["offset"] = map[string]any{"uri": trackURI}
	}
	return c.put(ctx, "/me/player/play", nil, body)
}
```

`internal/auth/spotifyauth.go:21` : ajouter ` user-modify-playback-state` à `scopes`.

`internal/app/service.go` : champs `openLink func(string) error` et `spotifyPlay func(ctx context.Context, contextURI, trackURI string, shuffle bool) error`, initialisés dans `New` à `openURL` et à `nil` ; `SetCatalog("spotify", c)` affecte `s.spotifyPlay = w.Play` quand `c` est un `*catalog.SpotifyWeb`. Dans `PlayFrom`, juste après le cas `*player.Spotify` (Mac, inchangé), ajouter :

```go
	if source == "spotify" && s.spotifyPlay != nil {
		// Hors Mac : lecture par l'API web ; refusée (pas Premium, aucun appareil) → l'album s'ouvre dans Spotify.
		err := s.spotifyPlay(ctx, collectionID, trackID, shuffle)
		if errors.Is(err, catalog.ErrPlaybackRefused) {
			return s.openLink(collectionID)
		}
		return err
	}
```

Dans `Queue()`, le cas `*player.Spotify` (file par l'API web) doit aussi s'appliquer quand la source active est « spotify » sans lecteur AppleScript : remplacer `case *player.Spotify:` par un test `sourceOf(m) == "spotify"` placé avant le `switch`, avec le même corps (Apple Music sous Windows : `q.Note = "File indisponible"`). Test à ajouter dans `queue_test.go` : un lecteur factice nommé « spotify » sans connexion Spotify → `Note == "Connecte Spotify pour voir la suite"`.

Dans `PlayItem`, si `source == "spotify"` et que le lecteur n'est pas un `*player.Spotify` : même logique avec `contextURI = id`, `trackURI = ""` (ou, si `id` est un titre `spotify:track:`, `PUT /me/player/play` avec `{"uris": [id]}` — ajouter `func (c *SpotifyWeb) PlayTrack(ctx, trackURI string) error` sur le modèle de `Play`).

- [ ] **Step 4: Run tests and builds**

Run: `CGO_ENABLED=1 go test -race . ./internal/... && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -o /tmp/vinyl.exe .`
Expected: tout passe.

- [ ] **Step 5: Commit**

```bash
git add internal/catalog internal/auth internal/app
git commit -m "feat(windows): lancer un album Spotify par l'API web, repli sur le lien"
```

---

### Task 9: Intégration continue et release automatiques

**Files:**
- Create: `.github/workflows/ci.yml`, `.github/workflows/release.yml`
- Modify: `build/windows/info.json` (version), `build/windows/nsis/project.nsi` (nom de l'installateur si besoin)

**Interfaces:**
- Consumes: tâches précédentes (la compilation Windows doit passer).
- Produces: release GitHub avec `Vinyl-mac.zip` et `Vinyl-Setup.exe` à chaque tag `v*`.

- [ ] **Step 1: Write the CI workflow** — `.github/workflows/ci.yml`

```yaml
name: CI
on:
  push:
    branches: [main, "feat/**"]
  pull_request:
jobs:
  test:
    strategy:
      matrix:
        os: [macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.26" }
      - uses: actions/setup-node@v4
        with: { node-version: "24" }
      - name: Frontend
        working-directory: frontend
        run: npm ci && npx tsc --noEmit && npm test && npm run build
      - name: Go (Mac)
        if: runner.os == 'macOS'
        env: { CGO_ENABLED: "1" }
        run: go vet . ./internal/... && go test -race . ./internal/...
      - name: Go (Windows)
        if: runner.os == 'Windows'
        env: { CGO_ENABLED: "0" }
        run: go vet . ./internal/... && go test . ./internal/...
```

- [ ] **Step 2: Write the release workflow** — `.github/workflows/release.yml`

```yaml
name: Release
on:
  push:
    tags: ["v*"]
permissions:
  contents: write
jobs:
  mac:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.26" }
      - uses: actions/setup-node@v4
        with: { node-version: "24" }
      - run: go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29
      - run: CGO_ENABLED=1 wails3 package
      - run: ditto -c -k --keepParent bin/vinyl.app Vinyl-mac.zip
      - uses: actions/upload-artifact@v4
        with: { name: mac, path: Vinyl-mac.zip }
  windows:
    runs-on: windows-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.26" }
      - uses: actions/setup-node@v4
        with: { node-version: "24" }
      - run: choco install nsis -y
      - run: go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.29
      - run: wails3 package
        env: { CGO_ENABLED: "0" }
      - name: Renommer l'installateur
        shell: pwsh
        run: Get-ChildItem bin -Filter *installer*.exe | Select-Object -First 1 | Copy-Item -Destination Vinyl-Setup.exe
      - uses: actions/upload-artifact@v4
        with: { name: windows, path: Vinyl-Setup.exe }
  release:
    needs: [mac, windows]
    runs-on: ubuntu-latest
    steps:
      - uses: actions/download-artifact@v4
      - uses: softprops/action-gh-release@v2
        with:
          files: |
            mac/Vinyl-mac.zip
            windows/Vinyl-Setup.exe
          body: |
            **Mac (Apple Silicon)** : `Vinyl-mac.zip` — app non signée : clic droit → Ouvrir au premier lancement.
            **Windows 10/11** : `Vinyl-Setup.exe` — non signé : « Informations complémentaires » → « Exécuter quand même ».
```

- [ ] **Step 3: Validate locally what can be**

Run: `ruby -ryaml -e 'YAML.load_file(".github/workflows/ci.yml"); YAML.load_file(".github/workflows/release.yml"); puts "ok"'`
Expected: `ok`. Le nom du fichier produit par `wails3 package` sous Windows est à confirmer au premier passage de la CI (étape « Renommer l'installateur »).

- [ ] **Step 4: Commit**

```bash
git add .github build/windows
git commit -m "ci: tests Mac et Windows, release automatique à chaque tag"
```

---

### Task 10: Vérification sur le PC Windows

**Files:**
- Create: `docs/windows-verification.md`

- [ ] **Step 1: Write the checklist** — `docs/windows-verification.md`

```markdown
# Vérification de Vinyl sous Windows

Installer `Vinyl-Setup.exe` (artefact de la CI ou de la release), lancer Vinyl.

Avec **Spotify** (app de bureau ou du Microsoft Store), puis avec **Apple Music pour Windows** :

1. Lancer un morceau dans l'app : la platine affiche titre, artiste, pochette ; le disque tourne ; le bras avance.
2. Touches Vinyl : lecture/pause, suivant, précédent ; glisser le bras plus loin : la lecture avance.
3. Aléatoire : le témoin suit l'état de l'app.
4. Écouter trois morceaux : la pile se remplit (pochettes visibles).
5. Cliquer sur le 2ᵉ disque de la pile : il rejoue ; à sa fin, retour au morceau interrompu, à la bonne position.
6. Spotify seulement : étagère (albums, playlists), page d'un album, « Lecture » : la lecture démarre
   (Premium) ou l'album s'ouvre dans Spotify (sans Premium).
7. Apple Music : l'étagère affiche « La bibliothèque Apple Music n'est disponible que sur Mac. »
8. Fermer l'app : pas d'erreur ; désinstaller depuis Paramètres → Applications : Vinyl disparaît.

Pour chaque point qui échoue : noter ce qui s'affiche, et joindre une capture d'écran.
```

- [ ] **Step 2: Commit**

```bash
git add docs/windows-verification.md
git commit -m "docs: liste de vérification de Vinyl sous Windows"
```

- [ ] **Step 3: Hand off to the user**

Pousser `feat/windows` (avec l'accord de l'utilisateur), récupérer l'installateur produit par la CI, et faire dérouler la liste par l'utilisateur sur son PC. Corriger chaque point en échec (débogage systématique : reproduire, cause, test, correction).
