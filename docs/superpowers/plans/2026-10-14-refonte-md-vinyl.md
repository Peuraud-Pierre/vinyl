# Refonte « comme MD Vinyl » — Plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ajouter à l'app Vinyl la page album, la file d'attente, le compteur d'écoutes, l'étagère coverflow et le partage story, et finir la platine, pour coller aux écrans de MD Vinyl dans `screen/`.

**Architecture:** Le backend Go gagne un paquet `internal/stats`. Les lecteurs (`internal/player`) reçoivent des capacités optionnelles via des interfaces vérifiées par assertion de type : modes de lecture, lecture depuis un titre, titres à venir. `Watcher` et `Player` gardent ainsi leurs signatures. Le catalogue gagne `Tracks`. Le service les expose au frontend. Côté frontend, toute la logique calculable (durées, positions coverflow, rouleaux) va dans des modules purs testés sous Vitest. Les vues (`views/album.ts`, `shell/queue.ts`, `share/story.ts`) sont branchées dans le routeur et le cadre existants.

**Tech Stack:** Go 1.26, Wails v3.0.0-beta.29, TypeScript vanilla, Tailwind v4, Vitest, AppleScript via `osascript`.

**Spec:** `docs/superpowers/specs/2026-10-14-refonte-md-vinyl-design.md`

## Global Constraints

- Branche : `feat/refonte-graphique`.
- Commandes Go : `CGO_ENABLED=1 go test . ./internal/...` et `go vet . ./internal/...` (jamais `./...`, à cause des dossiers mobiles générés).
- Bindings : `wails3 generate bindings -clean=true -ts -i` après chaque changement de signature du service.
- AppleScript : valeurs utilisateur uniquement via argv, dans `on run argv` ; toujours tester `application id "…" is running` avant un `tell` ; ne jamais lancer Musique ou Spotify.
- Bundle IDs : Musique `com.apple.Music`, Spotify `com.spotify.client`.
- Séparateurs : champs `\x1f` (`player.Sep`), enregistrements `\x1e`.
- Réels AppleScript lus avec `parseSeconds` / `parseMillis` (virgule décimale acceptée).
- Une variable AppleScript ne doit jamais s'appeler `st`, `nd`, `rd` ni `th` (ordinaux réservés).
- File d'attente : au plus 30 titres à venir ; playlists Spotify lues jusqu'à 500 titres.
- Stats : `~/Library/Application Support/Vinyl/stats.json`, date au format `AAAA.MM.JJ`.
- Story : canvas 1080×1920.
- Coverflow : ±70°, 64 px entre tranches, perspective 1300 px, fenêtre DOM ±12, un cran de molette tous les 60 px.
- Interface en français ; échapper toute donnée externe avec `escapeHtml` et `cssUrl`.
- Commits : `type(portée): description` en français, terminés par `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>`.

## Review Focus

1. **Album Musique sans numéros de piste ou avec plusieurs disques** : la page doit trier par disque puis piste, les pistes sans numéro (0) en dernier, sans doublons (test Task 4).
2. **Titre ou album contenant guillemets, accents ou « - » en tête** : aucun échec AppleScript lors de `Tracks` ou `PlayFrom` (passage par argv, testé Task 4 et Task 5).
3. **File d'attente avec le dernier titre de la playlist en cours** : liste vide, sans erreur, en répétition « off » ; reprise au début en répétition « all » (test Task 7).
4. **Durées extrêmes** : 0, moins d'une minute, plus d'une heure, valeur non finie (`NaN`) ; le format ne doit jamais afficher `NaN′` (test Task 6).
5. **Compteur au-delà de 8 chiffres ou négatif** : le compteur à rouleaux affiche 99999999 au maximum et 0 au minimum (test Task 3).

---

## Structure des fichiers

```
internal/
  stats/stats.go, stats_test.go              # Task 2
  player/watcher.go (+ SetOnPlaying)         # Task 2
  player/player.go  (+ Track, RepeatMode, interfaces optionnelles)  # Task 4, 7
  player/applemusic.go (+ PlayList, PlayInPlaylist, modes, Upcoming) # Task 5, 7
  player/spotify.go (+ PlayInContext, modes) # Task 5, 7
  catalog/catalog.go (+ Collection, Tracks)  # Task 4
  catalog/musiclibrary.go (+ Tracks)         # Task 4
  catalog/spotifyweb.go (+ Tracks, Queue)    # Task 4, 7
  app/service.go (+ Stats, Tracks, PlayFrom, Queue, SetShuffle, SetRepeat)  # Task 2, 4, 5, 7
main.go                                      # Task 2
frontend/src/
  turntable/label.ts (+ labelColorFor)       # Task 1
  turntable/arm.ts (ARM_REST = 0)            # Task 1
  turntable/turntable.ts                     # Task 1
  ui/odometer.ts, odometer.test.ts           # Task 3
  ui/format.ts, format.test.ts               # Task 6
  views/platine.ts (+ bande stats)           # Task 3
  views/album.ts                             # Task 6
  views/coverflow.ts, coverflow.test.ts      # Task 8
  views/bibliotheque.ts                      # Task 8
  shell/queue.ts                             # Task 9
  shell/shell.ts                             # Task 9
  share/story.ts                             # Task 10
  router.ts, main.ts, api.ts, ui/icons.ts    # Task 3, 6, 8, 9
```

---

### Task 1: Finitions de la platine (étiquette noire, bras droit au repos)

**Files:**
- Modify: `frontend/src/turntable/label.ts`, `frontend/src/turntable/turntable.ts`, `frontend/src/turntable/arm.ts`
- Test: `frontend/src/turntable/label.test.ts`, `frontend/src/turntable/arm.test.ts`

**Interfaces:**
- Produces: `labelColorFor(texture: string, tint: string | null): string` dans `label.ts`.

- [ ] **Step 1: Tests**

Ajouter à `frontend/src/turntable/label.test.ts` (import `labelColorFor` depuis `./label`) :

```ts
describe("couleur de l'étiquette", () => {
  test("texture noire : étiquette noire, quelle que soit la pochette", () => {
    expect(labelColorFor("black", "#c47a3a")).toBe("#0b0b0b");
  });
  test("picture disc : teinte de la pochette, crème par défaut", () => {
    expect(labelColorFor("picture", "#e9e3b0")).toBe("#e9e3b0");
    expect(labelColorFor("picture", null)).toBe("#efe6d2");
  });
  test("autres textures : noire", () => {
    for (const t of ["yolk", "split", "solid"]) expect(labelColorFor(t, "#123456")).toBe("#0b0b0b");
  });
});
```

Ajouter `describe, ` à l'import vitest du fichier s'il manque.

Dans `frontend/src/turntable/arm.test.ts`, ajouter :

```ts
test("au repos, le bras est vertical", () => {
  expect(ARM_REST).toBe(0);
});
```

Run: `cd frontend && npm test`
Expected : FAIL (`labelColorFor` absent ; `ARM_REST` vaut 4).

- [ ] **Step 2: Implémenter**

Dans `frontend/src/turntable/label.ts`, ajouter :

```ts
// Étiquette noire comme sur MD Vinyl ; en picture disc, elle prend la teinte de la pochette.
export function labelColorFor(texture: string, tint: string | null): string {
  if (texture === "picture") return tint ?? "#efe6d2";
  return "#0b0b0b";
}
```

Dans `frontend/src/turntable/arm.ts` : `export const ARM_REST = 0;`.

Dans `frontend/src/turntable/turntable.ts` :
- importer `labelColorFor` depuis `./label` ;
- remplacer le champ `private labelColor = "#0b0b0b";` par `private labelTint: string | null = null;` ;
- dans `loadLabelColor`, remplacer `this.labelColor = tint ?? "#0b0b0b";` par `this.labelTint = tint;` ;
- dans `paint()`, remplacer la dernière ligne par :

```ts
    const color = labelColorFor(tex, this.labelTint);
    this.label.innerHTML = labelMarkup({ title, artist: np?.artist ?? "", color, ring });
```

(En picture disc, l'étiquette reste visible, teintée, comme dans l'aperçu In Utero.)

- [ ] **Step 3: Vérifier**

Run: `cd frontend && npm test && npx tsc --noEmit`
Expected : PASS. Les tests de géométrie existants (« au repos, l'aiguille est hors du disque ») doivent toujours passer avec `ARM_REST = 0`.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/turntable
git commit -m "feat(platine): étiquette noire et bras droit au repos, comme MD Vinyl

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 2: Compteur d'écoutes côté Go

**Files:**
- Create: `internal/stats/stats.go`, `internal/stats/stats_test.go`
- Modify: `internal/player/watcher.go`, `internal/player/watcher_test.go`, `internal/app/service.go`, `main.go`

**Interfaces:**
- Produces :
  - `stats.Snapshot{Count int64 \`json:"count"\`; Since string \`json:"since"\`}`
  - `stats.NewStore(path string, now func() time.Time) *stats.Store`, `(*Store).Load()`, `(*Store).Get() Snapshot`, `(*Store).Played(trackID string) (Snapshot, bool)`
  - `(*player.Watcher).SetOnPlaying(func(player.NowPlaying))`
  - service : `Stats() stats.Snapshot` ; événement Wails `"stats"` (données `stats.Snapshot`)

- [ ] **Step 1: Tests du store**

`internal/stats/stats_test.go` :

```go
package stats

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

var day = func() time.Time { return time.Date(2026, 10, 14, 15, 0, 0, 0, time.Local) }

func TestFirstLaunchStartsAtZeroToday(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "stats.json"), day)
	s.Load()
	if g := s.Get(); g.Count != 0 || g.Since != "2026.10.14" {
		t.Errorf("got %+v", g)
	}
}

func TestPlayedCountsNewTracksOnlyAndPersists(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	s := NewStore(path, day)
	s.Load()
	if _, changed := s.Played("A"); !changed {
		t.Fatal("premier morceau : doit compter")
	}
	if _, changed := s.Played("A"); changed {
		t.Error("même morceau (pause puis reprise) : ne doit pas recompter")
	}
	if _, changed := s.Played(""); changed {
		t.Error("ID vide : ne compte pas")
	}
	got, _ := s.Played("B")
	if got.Count != 2 {
		t.Errorf("count = %d", got.Count)
	}
	s2 := NewStore(path, func() time.Time { return day().AddDate(0, 0, 3) })
	s2.Load()
	if g := s2.Get(); g.Count != 2 || g.Since != "2026.10.14" {
		t.Errorf("relu depuis le disque : %+v", g)
	}
}

func TestCorruptFileRestartsAtZero(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stats.json")
	os.WriteFile(path, []byte("{oups"), 0o644)
	s := NewStore(path, day)
	s.Load()
	if g := s.Get(); g.Count != 0 || g.Since != "2026.10.14" {
		t.Errorf("got %+v", g)
	}
}

func TestConcurrentPlaysAreAllCounted(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "stats.json"), day)
	s.Load()
	var wg sync.WaitGroup
	for i := range 20 {
		wg.Add(1)
		go func() { defer wg.Done(); s.Played(string(rune('a' + i))) }()
	}
	wg.Wait()
	// Les ID se succèdent tous différents du précédent, sauf collision d'ordre : au moins 1, au plus 20.
	if g := s.Get(); g.Count < 1 || g.Count > 20 {
		t.Errorf("count = %d", g.Count)
	}
}
```

Run: `CGO_ENABLED=1 go test ./internal/stats/` → Expected : FAIL (paquet vide).

- [ ] **Step 2: Implémenter le store**

`internal/stats/stats.go` :

```go
// Package stats compte les morceaux lus depuis le premier lancement.
package stats

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Snapshot struct {
	Count int64  `json:"count"`
	Since string `json:"since"` // AAAA.MM.JJ
}

type Store struct {
	path string
	now  func() time.Time
	mu   sync.Mutex
	cur  Snapshot
	last string // dernier morceau compté
}

func NewStore(path string, now func() time.Time) *Store { return &Store{path: path, now: now} }

func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "Vinyl", "stats.json"), nil
}

// Load lit le fichier ; absent ou illisible → 0 depuis aujourd'hui.
func (s *Store) Load() {
	s.mu.Lock()
	defer s.mu.Unlock()
	var v Snapshot
	if data, err := os.ReadFile(s.path); err == nil && json.Unmarshal(data, &v) == nil && v.Since != "" && v.Count >= 0 {
		s.cur = v
		return
	}
	s.cur = Snapshot{Since: s.now().Format("2006.01.02")}
	s.writeLocked()
}

func (s *Store) Get() Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cur
}

// Played compte trackID s'il diffère du dernier morceau compté. Renvoie l'état et s'il a changé.
func (s *Store) Played(trackID string) (Snapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if trackID == "" || trackID == s.last {
		return s.cur, false
	}
	s.last = trackID
	s.cur.Count++
	s.writeLocked()
	return s.cur, true
}

func (s *Store) writeLocked() {
	data, _ := json.MarshalIndent(s.cur, "", "  ")
	if os.MkdirAll(filepath.Dir(s.path), 0o755) != nil {
		return
	}
	tmp := s.path + ".tmp"
	if os.WriteFile(tmp, data, 0o644) == nil {
		os.Rename(tmp, s.path)
	}
}
```

Run: `CGO_ENABLED=1 go test -race ./internal/stats/` → Expected : PASS.

- [ ] **Step 3: Test du crochet du watcher**

Dans `internal/player/watcher_test.go`, ajouter :

```go
func TestOnPlayingCalledOnlyWhilePlaying(t *testing.T) {
	h := newHarness()
	var seen []string
	h.w.SetOnPlaying(func(np NowPlaying) { seen = append(seen, np.TrackID) })
	h.am.err, h.am.np = nil, NowPlaying{TrackID: "a", State: StatePaused}
	h.tick(0)
	h.am.np.State = StatePlaying
	h.tick(500 * time.Millisecond)
	h.am.np.TrackID = "b"
	h.tick(500 * time.Millisecond)
	if len(seen) != 2 || seen[0] != "a" || seen[1] != "b" {
		t.Errorf("appels = %v", seen)
	}
}
```

Run: `CGO_ENABLED=1 go test ./internal/player/ -run OnPlaying` → Expected : FAIL (`SetOnPlaying` indéfini).

- [ ] **Step 4: Implémenter le crochet**

Dans `internal/player/watcher.go` :
- ajouter le champ `onPlaying func(NowPlaying)` au struct `Watcher` (avec les champs « utilisés uniquement depuis Tick ») ;
- ajouter la méthode :

```go
// SetOnPlaying enregistre un rappel appelé à chaque sondage où la source active joue (à brancher avant Run).
func (w *Watcher) SetOnPlaying(f func(NowPlaying)) { w.onPlaying = f }
```

- à la fin de `Tick`, juste avant `if send { w.emit(snap) }`, ajouter :

```go
	if w.onPlaying != nil && snap.Problem == "" && snap.NowPlaying.State == StatePlaying {
		w.onPlaying(snap.NowPlaying)
	}
```

Le test attend un appel par tick en lecture : `seen` vaut `["a", "b"]` car il y a deux ticks en lecture. Le dédoublonnage est fait par `stats.Played`.

Run: `CGO_ENABLED=1 go test -race ./internal/player/` → Expected : PASS.

- [ ] **Step 5: Brancher dans le service et `main.go`**

Dans `internal/app/service.go` : import `"vinyl/internal/stats"`, ajouter le champ `stats *stats.Store` au struct, puis :

```go
func (s *VinylService) SetStats(st *stats.Store) { s.stats = st }

// Stats renvoie le compteur « Morceaux lus depuis ».
func (s *VinylService) Stats() stats.Snapshot {
	if s.stats == nil {
		return stats.Snapshot{}
	}
	return s.stats.Get()
}
```

Dans `main.go`, après la création de `service` :

```go
	statsPath, err := stats.DefaultPath()
	if err != nil {
		log.Fatal(err)
	}
	statStore := stats.NewStore(statsPath, time.Now)
	statStore.Load()
	service.SetStats(statStore)
	watcher.SetOnPlaying(func(np player.NowPlaying) {
		if np.TrackID == "" {
			return // radio ou flux sans identifiant : pas compté
		}
		if snap, changed := statStore.Played(np.Source + ":" + np.TrackID); changed {
			wailsApp.Event.Emit("stats", snap)
		}
	})
```

Ajouter dans `init()` : `application.RegisterEvent[stats.Snapshot]("stats")`, et l'import `"vinyl/internal/stats"`.

Run:
```bash
CGO_ENABLED=1 go build -o /dev/null . && CGO_ENABLED=1 go vet . ./internal/... && CGO_ENABLED=1 go test . ./internal/...
wails3 generate bindings -clean=true -ts -i
```
Expected : tout passe ; `frontend/bindings/vinyl/internal/stats/` existe.

- [ ] **Step 6: Commit**

```bash
git add -A internal main.go frontend/bindings
git commit -m "feat(stats): compteur de morceaux lus, enregistré et diffusé

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 3: Bande « Morceaux lus depuis » avec compteur à rouleaux

**Files:**
- Create: `frontend/src/ui/odometer.ts`, `frontend/src/ui/odometer.test.ts`
- Modify: `frontend/src/views/platine.ts`, `frontend/src/api.ts`, `frontend/src/styles.css`

**Interfaces:**
- Consumes: service `Stats()`, événement `"stats"` (Task 2).
- Produces : `odometerDigits(n: number, width?: number): number[]` ; `mountOdometer(el: HTMLElement, width?: number): (n: number) => void` ; dans `api.ts`, `StatsSnapshot { count: number; since: string }` et `onStats(cb)`.

- [ ] **Step 1: Tests**

`frontend/src/ui/odometer.test.ts` :

```ts
import { expect, test } from "vitest";
import { odometerDigits } from "./odometer";

test("8 chiffres avec zéros à gauche", () => {
  expect(odometerDigits(34)).toEqual([0, 0, 0, 0, 0, 0, 3, 4]);
});
test("bornes : négatif → 0, trop grand → 99999999", () => {
  expect(odometerDigits(-5)).toEqual([0, 0, 0, 0, 0, 0, 0, 0]);
  expect(odometerDigits(1234567890)).toEqual([9, 9, 9, 9, 9, 9, 9, 9]);
});
test("non entier ou NaN", () => {
  expect(odometerDigits(12.9)).toEqual([0, 0, 0, 0, 0, 0, 1, 2]);
  expect(odometerDigits(NaN)).toEqual([0, 0, 0, 0, 0, 0, 0, 0]);
});
test("largeur personnalisée", () => {
  expect(odometerDigits(7, 3)).toEqual([0, 0, 7]);
});
```

Run: `cd frontend && npm test` → Expected : FAIL.

- [ ] **Step 2: Implémenter le compteur**

`frontend/src/ui/odometer.ts` :

```ts
// Compteur à rouleaux : chaque chiffre est une colonne 0–9 qui défile verticalement.
export function odometerDigits(n: number, width = 8): number[] {
  const max = 10 ** width - 1;
  const v = Number.isFinite(n) ? Math.min(max, Math.max(0, Math.floor(n))) : 0;
  return String(v).padStart(width, "0").split("").map(Number);
}

export function mountOdometer(el: HTMLElement, width = 8): (n: number) => void {
  const reel = Array.from({ length: 10 }, (_, d) => `<span>${d}</span>`).join("");
  el.className = "vy-odo";
  el.innerHTML = Array.from({ length: width }, () => `<div class="vy-odo-cell"><div class="vy-odo-reel">${reel}</div></div>`).join("");
  const reels = [...el.querySelectorAll<HTMLElement>(".vy-odo-reel")];
  return (n) => odometerDigits(n, width).forEach((d, i) => { reels[i]!.style.transform = `translateY(${-d}em)`; });
}
```

Ajouter à `frontend/src/styles.css` :

```css
/* Compteur à rouleaux : cases noires, chiffre précédent et suivant à demi visibles. */
.vy-odo { display: inline-flex; gap: 10px; padding: 12px 14px; border-radius: 22px; background: #111; box-shadow: inset 0 2px 6px rgba(0,0,0,.6), 0 1px 0 rgba(255,255,255,.05); font-size: 30px; }
.vy-odo-cell { width: 1.35em; height: 1.6em; overflow: hidden; border-radius: 10px; background: #1d1d1d; box-shadow: inset 0 0 0 1px rgba(255,255,255,.08); color: #f4f4f4; text-align: center; -webkit-mask-image: linear-gradient(transparent, #000 30%, #000 70%, transparent); mask-image: linear-gradient(transparent, #000 30%, #000 70%, transparent); }
.vy-odo-reel { display: flex; flex-direction: column; line-height: 1em; padding-top: .3em; transition: transform .6s cubic-bezier(.2,.8,.2,1); }
.vy-odo-reel span { height: 1em; }
```

Run: `cd frontend && npm test` → Expected : PASS.

- [ ] **Step 3: Accès aux stats depuis `api.ts`**

Ajouter à `frontend/src/api.ts` :

```ts
export interface StatsSnapshot { count: number; since: string; }
export function onStats(cb: (s: StatsSnapshot) => void): () => void {
  return Events.On("stats", (ev) => cb(ev.data as StatsSnapshot));
}
```

- [ ] **Step 4: Bande sous la platine**

Dans `frontend/src/views/platine.ts` :
- La `<section class="relative h-screen">` devient le premier enfant d'un conteneur défilant. Remplacer l'ouverture du template par `<div id="pl-scroll" class="h-screen overflow-y-auto shelf-scroll"><section class="relative h-screen">`.
- Après `</section>`, ajouter :

```html
      <footer class="flex flex-col items-center gap-6 py-16" style="background:#1b1b1b">
        <div id="pl-odo"></div>
        <button id="pl-stats-share" class="flex items-center gap-3 text-[20px] text-white/90 hover:text-white">
          <span id="pl-since">Morceaux lus</span> ${icons.share}
        </button>
      </footer>
    </div>
```

- Dans `mount`, après la création de `tt` :

```ts
  const setCount = mountOdometer($("pl-odo"));
  let statsNow: StatsSnapshot = { count: 0, since: "" };
  const showStats = (s: StatsSnapshot) => {
    statsNow = s;
    setCount(s.count);
    $("pl-since").textContent = s.since ? `Morceaux lus depuis ${s.since}` : "Morceaux lus";
  };
  const offStats = onStats(showStats);
  api.Stats().then(showStats).catch(() => {});
  $("pl-stats-share").onclick = () => openShareCard({ title: `${statsNow.count} morceaux lus`, subtitle: statsNow.since ? `depuis ${statsNow.since}` : "", artworkUrl: snap?.nowPlaying.artworkUrl ?? "" });
```

- Ajouter `offStats()` dans la fonction de nettoyage renvoyée.
- Imports : `mountOdometer` depuis `../ui/odometer` ; `onStats`, `type StatsSnapshot` depuis `../api`.
- `ResizeObserver` observe toujours `pl-wrap` (hauteur d'écran), la scène reste à l'échelle.

Run: `cd frontend && npm test && npx tsc --noEmit` → Expected : PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src
git commit -m "feat(platine): bande « Morceaux lus depuis » avec compteur à rouleaux

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 4: `Tracks` — titres d'un album ou d'une playlist

**Files:**
- Modify: `internal/player/player.go`, `internal/catalog/catalog.go`, `internal/catalog/musiclibrary.go`, `internal/catalog/spotifyweb.go`, `internal/app/service.go`
- Test: `internal/catalog/musiclibrary_test.go`, `internal/catalog/spotifyweb_test.go`, `internal/catalog/live_test.go`

**Interfaces:**
- Produces :
  - `player.Track{ID string; Number int; Title, Artist string; DurationMs int64; Loved bool}` (tags JSON `id, number, title, artist, durationMs, loved`)
  - `catalog.Collection{ID, Kind, Title, Artist, Genre, Year, ArtworkURL, Source string; Tracks []player.Track}` (tags `id, kind, title, artist, genre, year, artworkUrl, source, tracks`)
  - `Catalog.Tracks(ctx, id string) (Collection, error)` ajoutée à l'interface
  - service `Tracks(source, id string) (catalog.Collection, error)`

- [ ] **Step 1: Types**

Ajouter à `internal/player/player.go` :

```go
// Track est un titre d'album, de playlist ou de file d'attente.
type Track struct {
	ID         string `json:"id"` // persistent ID (Musique) ou URI spotify:track:…
	Number     int    `json:"number"`
	Title      string `json:"title"`
	Artist     string `json:"artist"`
	DurationMs int64  `json:"durationMs"`
	Loved      bool   `json:"loved"`
}

// ParseSeconds est exporté pour les autres paquets (catalog) : secondes AppleScript → ms.
func ParseSeconds(s string) int64 { return parseSeconds(s) }
```

Ajouter à `internal/catalog/catalog.go` (import `"vinyl/internal/player"`) :

```go
type Collection struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"` // album | playlist
	Title      string         `json:"title"`
	Artist     string         `json:"artist"`
	Genre      string         `json:"genre"`
	Year       string         `json:"year"`
	ArtworkURL string         `json:"artworkUrl"`
	Source     string         `json:"source"`
	Tracks     []player.Track `json:"tracks"`
}
```

et la méthode `Tracks(ctx context.Context, id string) (Collection, error)` dans l'interface `Catalog`.

- [ ] **Step 2: Tests Musique**

Ajouter à `internal/catalog/musiclibrary_test.go` :

```go
func rec(fields ...string) string { return strings.Join(fields, player.Sep) + rs }

func TestMusicAlbumTracksSortedByDiscThenNumber(t *testing.T) {
	// id, disque, piste, titre, artiste, durée, aimé, genre, année
	out := rec("C", "2", "1", "Disque 2 piste 1", "Nirvana", "120,5", "0", "Alternative", "1993") +
		rec("B", "1", "2", `Heart-Shaped "Box"`, "Nirvana", "281", "1", "Alternative", "1993") +
		rec("Z", "1", "0", "Sans numéro", "Nirvana", "60", "0", "Alternative", "1993") +
		rec("A", "1", "1", "Serve the Servants", "Nirvana", "217", "0", "Alternative", "1993")
	r := &fakeRunner{out: []string{out}}
	c, err := NewMusicLibrary(r, nil, t.TempDir()).Tracks(context.Background(), "album:In Utero"+player.Sep+"Nirvana")
	if err != nil {
		t.Fatal(err)
	}
	var ids []string
	for _, tr := range c.Tracks {
		ids = append(ids, tr.ID)
	}
	if strings.Join(ids, ",") != "A,B,Z,C" {
		t.Errorf("ordre = %v (disque, puis piste, sans numéro en fin de disque)", ids)
	}
	if c.Title != "In Utero" || c.Artist != "Nirvana" || c.Genre != "Alternative" || c.Year != "1993" || c.Kind != "album" {
		t.Errorf("en-tête = %+v", c)
	}
	if !c.Tracks[1].Loved || c.Tracks[1].DurationMs != 281000 || c.Tracks[3].DurationMs != 120500 {
		t.Errorf("titres = %+v", c.Tracks)
	}
	if got := r.calls[0][1:]; strings.Join(got, "|") != "In Utero|Nirvana" {
		t.Errorf("argv = %q", got)
	}
}

func TestMusicPlaylistTracksKeepOrderAndName(t *testing.T) {
	out := rec("Ma playlist") + rec("Y", "0", "7", "Deux", "B", "200", "0", "", "0") + rec("X", "0", "3", "Un", "A", "100", "0", "", "0")
	c, err := NewMusicLibrary(&fakeRunner{out: []string{out}}, nil, t.TempDir()).Tracks(context.Background(), "playlist:P1")
	if err != nil {
		t.Fatal(err)
	}
	if c.Title != "Ma playlist" || c.Kind != "playlist" || len(c.Tracks) != 2 || c.Tracks[0].ID != "Y" || c.Tracks[0].Number != 1 || c.Year != "" {
		t.Errorf("got %+v", c)
	}
}

func TestMusicTracksNotRunningAndBadID(t *testing.T) {
	m := NewMusicLibrary(&fakeRunner{out: []string{"NOTRUNNING"}}, nil, t.TempDir())
	if _, err := m.Tracks(context.Background(), "album:X"+player.Sep+"Y"); err != player.ErrNotRunning {
		t.Errorf("got %v", err)
	}
	if _, err := m.Tracks(context.Background(), "inconnu"); err == nil {
		t.Error("ID invalide → erreur")
	}
}
```

Run: `CGO_ENABLED=1 go test ./internal/catalog/ -run Tracks` → Expected : FAIL (compilation : `Tracks` indéfinie ; SpotifyWeb n'implémente plus `Catalog`).

- [ ] **Step 3: Implémenter `Tracks` côté Musique**

Ajouter à `internal/catalog/musiclibrary.go` :

```go
// Champs : persistent ID, disque, piste, titre, artiste, durée, aimé (1/0), genre, année.
const trackFields = `(persistent ID of t) & us & ((disc number of t) as string) & us & ((track number of t) as string) & us & (name of t) & us & (artist of t) & us & ((duration of t) as string) & us & lv & us & (genre of t) & us & ((year of t) as string) & rs`

const lovedSnippet = `
		set lv to "0"
		try
			if favorited of t then set lv to "1"
		on error
			try
				if loved of t then set lv to "1"
			end try
		end try`

const albumTracksScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	set us to character id 31
	set rs to character id 30
	tell application id "com.apple.Music"
		set ts to (every track of library playlist 1 whose album is (item 1 of argv) and album artist is (item 2 of argv))
		if (count of ts) is 0 then set ts to (every track of library playlist 1 whose album is (item 1 of argv))
		set out to ""
		repeat with t in ts` + lovedSnippet + `
			set out to out & ` + trackFields + `
		end repeat
		return out
	end tell
end run`

const playlistTracksScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	set us to character id 31
	set rs to character id 30
	tell application id "com.apple.Music"
		set p to (first user playlist whose persistent ID is (item 1 of argv))
		set out to (name of p) & rs
		repeat with t in (tracks of p)` + lovedSnippet + `
			set out to out & ` + trackFields + `
		end repeat
		return out
	end tell
end run`

func parseTrackRecord(r []string) (player.Track, int, string, string, bool) {
	if len(r) != 9 {
		return player.Track{}, 0, "", "", false
	}
	disc, _ := strconv.Atoi(strings.TrimSpace(r[1]))
	num, _ := strconv.Atoi(strings.TrimSpace(r[2]))
	year := strings.TrimSpace(r[8])
	if year == "0" {
		year = ""
	}
	return player.Track{ID: r[0], Number: num, Title: r[3], Artist: r[4], DurationMs: player.ParseSeconds(r[5]), Loved: r[6] == "1"}, disc, strings.TrimSpace(r[7]), year, true
}

func (m *MusicLibrary) Tracks(ctx context.Context, id string) (Collection, error) {
	kind, rest, ok := strings.Cut(id, ":")
	if !ok || (kind != "album" && kind != "playlist") {
		return Collection{}, fmt.Errorf("musique : ID de collection invalide %q", id)
	}
	c := Collection{ID: id, Kind: kind, Source: "applemusic"}
	var recs [][]string
	var err error
	if kind == "album" {
		album, artist, _ := strings.Cut(rest, player.Sep)
		c.Title, c.Artist = album, artist
		recs, err = m.recordsArgs(ctx, albumTracksScript, album, artist)
	} else {
		recs, err = m.recordsArgs(ctx, playlistTracksScript, rest)
		if err == nil && len(recs) > 0 {
			c.Title = recs[0][0]
			recs = recs[1:]
		}
	}
	if err != nil {
		return Collection{}, err
	}
	type keyed struct {
		t    player.Track
		disc int
	}
	var rows []keyed
	for _, r := range recs {
		t, disc, genre, year, ok := parseTrackRecord(r)
		if !ok {
			continue
		}
		if c.Genre == "" {
			c.Genre = genre
		}
		if c.Year == "" {
			c.Year = year
		}
		if c.Artist == "" {
			c.Artist = t.Artist
		}
		rows = append(rows, keyed{t, disc})
	}
	if kind == "album" {
		// Disque puis piste ; une piste sans numéro (0) va en fin de son disque.
		order := func(n int) int {
			if n <= 0 {
				return 1 << 30
			}
			return n
		}
		sort.SliceStable(rows, func(i, j int) bool {
			if rows[i].disc != rows[j].disc {
				return rows[i].disc < rows[j].disc
			}
			return order(rows[i].t.Number) < order(rows[j].t.Number)
		})
	}
	for i, r := range rows {
		if kind == "playlist" {
			r.t.Number = i + 1
		}
		c.Tracks = append(c.Tracks, r.t)
	}
	if url, err := m.Artwork(ctx, id); err == nil {
		c.ArtworkURL = url
	}
	return c, nil
}
```

Refactorer `records` pour accepter des arguments (garder l'ancien nom pour les appels existants) :

```go
func (m *MusicLibrary) records(ctx context.Context, script string) ([][]string, error) {
	return m.recordsArgs(ctx, script)
}

func (m *MusicLibrary) recordsArgs(ctx context.Context, script string, args ...string) ([][]string, error) {
	out, err := m.run.Run(ctx, script, args...)
	if err != nil {
		return nil, err
	}
	if out == "NOTRUNNING" {
		return nil, player.ErrNotRunning
	}
	var recs [][]string
	for _, r := range strings.Split(out, recordSep) {
		if r != "" {
			recs = append(recs, strings.Split(r, player.Sep))
		}
	}
	return recs, nil
}
```

Imports à ajouter : `"fmt"`, `"strconv"`.

**Attention** : `m.Artwork` appelle le runner (le faux runner renvoie `""` au 2ᵉ appel, ce qui est un « pas OK » ; c'est mémorisé et sans erreur). Les assertions des tests sur `r.calls[0]` restent valides.

- [ ] **Step 4: Tests Spotify**

Ajouter à `internal/catalog/spotifyweb_test.go` :

```go
func TestSpotifyAlbumTracksWithHearts(t *testing.T) {
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/albums/AL":
			w.Write([]byte(`{"name":"In Utero","release_date":"1993-09-21","genres":[],"artists":[{"name":"Nirvana"}],"images":[{"url":"https://img/u"}],
				"tracks":{"items":[{"uri":"spotify:track:1","name":"Serve the Servants","track_number":1,"duration_ms":217000,"artists":[{"name":"Nirvana"}]},
				{"uri":"spotify:track:2","name":"Scentless Apprentice","track_number":2,"duration_ms":228000,"artists":[{"name":"Nirvana"}]}]}}`))
		case "/me/tracks/contains":
			if r.URL.Query().Get("ids") != "1,2" {
				http.Error(w, "ids", 400)
				return
			}
			w.Write([]byte(`[false,true]`))
		default:
			http.Error(w, r.URL.Path, 404)
		}
	})
	col, err := c.Tracks(context.Background(), "spotify:album:AL")
	if err != nil {
		t.Fatal(err)
	}
	if col.Title != "In Utero" || col.Artist != "Nirvana" || col.Year != "1993" || col.ArtworkURL != "https://img/u" || len(col.Tracks) != 2 || !col.Tracks[1].Loved || col.Tracks[0].Loved {
		t.Errorf("got %+v", col)
	}
}

func TestSpotifyPlaylistTracksPaginated(t *testing.T) {
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/playlists/PL":
			w.Write([]byte(`{"name":"Mix","owner":{"display_name":"moi"},"images":[{"url":"https://img/p"}]}`))
		case r.URL.Path == "/playlists/PL/tracks" && r.URL.Query().Get("offset") == "0":
			w.Write([]byte(`{"items":[{"track":{"uri":"spotify:track:1","name":"A","duration_ms":1000,"artists":[{"name":"X"}]}}],"next":"more"}`))
		case r.URL.Path == "/playlists/PL/tracks" && r.URL.Query().Get("offset") == "100":
			w.Write([]byte(`{"items":[{"track":null},{"track":{"uri":"spotify:track:2","name":"B","duration_ms":2000,"artists":[{"name":"Y"}]}}],"next":null}`))
		case r.URL.Path == "/me/tracks/contains":
			w.Write([]byte(`[false,false]`))
		default:
			http.Error(w, r.URL.String(), 404)
		}
	})
	col, err := c.Tracks(context.Background(), "spotify:playlist:PL")
	if err != nil {
		t.Fatal(err)
	}
	if col.Title != "Mix" || col.Artist != "moi" || len(col.Tracks) != 2 || col.Tracks[1].Number != 2 || col.Tracks[1].Title != "B" {
		t.Errorf("got %+v", col)
	}
}
```

- [ ] **Step 5: Implémenter `Tracks` côté Spotify**

Ajouter à `internal/catalog/spotifyweb.go` :

```go
type spFullTrack struct {
	URI         string     `json:"uri"`
	Name        string     `json:"name"`
	TrackNumber int        `json:"track_number"`
	DurationMs  int64      `json:"duration_ms"`
	Artists     []spArtist `json:"artists"`
}

func (c *SpotifyWeb) Tracks(ctx context.Context, id string) (Collection, error) {
	switch {
	case strings.HasPrefix(id, "spotify:album:"):
		return c.albumTracks(ctx, id)
	case strings.HasPrefix(id, "spotify:playlist:"):
		return c.playlistTracks(ctx, id)
	}
	return Collection{}, fmt.Errorf("spotify : ID de collection invalide %q", id)
}

func toTrack(t spFullTrack, number int) player.Track {
	return player.Track{ID: t.URI, Number: number, Title: t.Name, Artist: artistNames(t.Artists), DurationMs: t.DurationMs}
}

func (c *SpotifyWeb) albumTracks(ctx context.Context, id string) (Collection, error) {
	var a struct {
		Name        string     `json:"name"`
		ReleaseDate string     `json:"release_date"`
		Genres      []string   `json:"genres"`
		Artists     []spArtist `json:"artists"`
		Images      []spImage  `json:"images"`
		Tracks      struct {
			Items []spFullTrack `json:"items"`
		} `json:"tracks"`
	}
	if err := c.get(ctx, "/albums/"+strings.TrimPrefix(id, "spotify:album:"), nil, &a); err != nil {
		return Collection{}, err
	}
	col := Collection{ID: id, Kind: "album", Title: a.Name, Artist: artistNames(a.Artists), Year: year(a.ReleaseDate), ArtworkURL: firstImage(a.Images), Source: "spotify"}
	if len(a.Genres) > 0 {
		col.Genre = a.Genres[0]
	}
	for _, t := range a.Tracks.Items {
		col.Tracks = append(col.Tracks, toTrack(t, t.TrackNumber))
	}
	c.markLoved(ctx, col.Tracks)
	return col, nil
}

func (c *SpotifyWeb) playlistTracks(ctx context.Context, id string) (Collection, error) {
	pid := strings.TrimPrefix(id, "spotify:playlist:")
	var p struct {
		Name   string    `json:"name"`
		Images []spImage `json:"images"`
		Owner  struct {
			DisplayName string `json:"display_name"`
		} `json:"owner"`
	}
	if err := c.get(ctx, "/playlists/"+pid, url.Values{"fields": {"name,images,owner(display_name)"}}, &p); err != nil {
		return Collection{}, err
	}
	col := Collection{ID: id, Kind: "playlist", Title: p.Name, Artist: p.Owner.DisplayName, ArtworkURL: firstImage(p.Images), Source: "spotify"}
	for offset := 0; offset < 500; offset += 100 {
		var page struct {
			Items []struct {
				Track *spFullTrack `json:"track"`
			} `json:"items"`
			Next *string `json:"next"`
		}
		q := url.Values{"limit": {"100"}, "offset": {strconv.Itoa(offset)}}
		if err := c.get(ctx, "/playlists/"+pid+"/tracks", q, &page); err != nil {
			return Collection{}, err
		}
		for _, it := range page.Items {
			if it.Track != nil && it.Track.URI != "" {
				col.Tracks = append(col.Tracks, toTrack(*it.Track, len(col.Tracks)+1))
			}
		}
		if page.Next == nil || *page.Next == "" {
			break
		}
	}
	c.markLoved(ctx, col.Tracks)
	return col, nil
}

// markLoved coche les titres likés (par lots de 50). Une erreur laisse simplement les cœurs vides.
func (c *SpotifyWeb) markLoved(ctx context.Context, ts []player.Track) {
	for start := 0; start < len(ts); start += 50 {
		end := min(start+50, len(ts))
		ids := make([]string, 0, end-start)
		for _, t := range ts[start:end] {
			ids = append(ids, strings.TrimPrefix(t.ID, "spotify:track:"))
		}
		var loved []bool
		if c.get(ctx, "/me/tracks/contains", url.Values{"ids": {strings.Join(ids, ",")}}, &loved) != nil {
			return
		}
		for i := range loved {
			if start+i < len(ts) {
				ts[start+i].Loved = loved[i]
			}
		}
	}
}
```

Import `"vinyl/internal/player"` dans `spotifyweb.go`. Note : `url.Values.Encode` encode la virgule de `ids` (`%2C`). Le serveur de test lit `r.URL.Query().Get("ids")`, qui la décode, donc le test reste valide.

Run: `CGO_ENABLED=1 go test -race ./internal/catalog/` → Expected : PASS.

- [ ] **Step 6: Service et test live**

Dans `internal/app/service.go` :

```go
// Tracks renvoie l'album ou la playlist avec ses titres.
func (s *VinylService) Tracks(source, id string) (catalog.Collection, error) {
	c, err := s.catalog(source)
	if err != nil {
		return catalog.Collection{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return c.Tracks(ctx, id)
}
```

Ajouter à `internal/catalog/live_test.go`, dans la boucle de `TestLiveMusicScriptsCompile` ou dans un nouveau test :

```go
func TestLiveMusicTracksScriptsCompile(t *testing.T) {
	if os.Getenv("VINYL_LIVE") == "" {
		t.Skip("définir VINYL_LIVE=1")
	}
	m := NewMusicLibrary(player.OSAScript{}, NewITunes(), t.TempDir())
	for _, id := range []string{"album:-e" + player.Sep + `"x"`, "playlist:0000000000000000"} {
		_, err := m.Tracks(context.Background(), id)
		if err != nil && !errors.Is(err, player.ErrNotAuthorized) && !errors.Is(err, player.ErrNotRunning) &&
			!strings.Contains(err.Error(), "(-1728)") && !strings.Contains(err.Error(), "(-1719)") {
			t.Errorf("%s : %v", id, err)
		}
	}
}
```

Ajouter les imports `"strings"`. Le code −1728 ou −1719, quand la playlist est introuvable, est accepté : seule une erreur de syntaxe ferait échouer le test.

Run:
```bash
CGO_ENABLED=1 go test . ./internal/... && CGO_ENABLED=1 go vet . ./internal/...
VINYL_LIVE=1 CGO_ENABLED=1 go test ./internal/catalog/ -run Live -v -count=1
wails3 generate bindings -clean=true -ts -i
```
Expected : PASS ; aucun « syntax error » dans la sortie live.

- [ ] **Step 7: Commit**

```bash
git add -A internal frontend/bindings
git commit -m "feat(catalog): titres d'un album ou d'une playlist (Musique et Spotify)

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 5: `PlayFrom` — lancer un album ou une playlist à partir d'un titre

**Files:**
- Modify: `internal/player/applemusic.go`, `internal/player/spotify.go`, `internal/app/service.go`
- Test: `internal/player/applemusic_test.go`, `internal/player/spotify_test.go`

**Interfaces:**
- Consumes: `catalog.Collection` (Task 4), `(*VinylService).Tracks` (Task 4).
- Produces :
  - `(*player.AppleMusic).PlayList(ctx, ids []string, startID string, shuffle bool) error`
  - `(*player.AppleMusic).PlayInPlaylist(ctx, playlistID, startID string, shuffle bool) error`
  - `(*player.Spotify).PlayInContext(ctx, contextURI, trackURI string, shuffle bool) error`
  - service `PlayFrom(source, collectionID, trackID string, shuffle bool) error`

- [ ] **Step 1: Tests**

Ajouter à `internal/player/applemusic_test.go` :

```go
func TestPlayListPassesIDsInOrderThroughArgv(t *testing.T) {
	r := &fakeRunner{}
	err := NewAppleMusic(r, t.TempDir()).PlayList(context.Background(), []string{"-e", "B", "C"}, "B", true)
	if err != nil {
		t.Fatal(err)
	}
	got := r.calls[0]
	if strings.Join(got.args, "|") != "B|1|-e|B|C" || !strings.Contains(got.script, "Vinyl — Album") {
		t.Errorf("args = %q", got.args)
	}
}

func TestPlayInPlaylistArgs(t *testing.T) {
	r := &fakeRunner{}
	NewAppleMusic(r, t.TempDir()).PlayInPlaylist(context.Background(), "P1", "", false)
	if strings.Join(r.calls[0].args, "|") != "P1||0" {
		t.Errorf("args = %q", r.calls[0].args)
	}
}
```

Ajouter à `internal/player/spotify_test.go` :

```go
func TestSpotifyPlayInContext(t *testing.T) {
	r := &fakeRunner{}
	s := NewSpotify(r)
	if err := s.PlayInContext(context.Background(), "spotify:album:9", "spotify:track:3", true); err != nil {
		t.Fatal(err)
	}
	if strings.Join(r.calls[0].args, "|") != "spotify:album:9|spotify:track:3|1" {
		t.Errorf("args = %q", r.calls[0].args)
	}
	if err := s.PlayInContext(context.Background(), "playlist:1", "", false); err == nil {
		t.Error("contexte non Spotify → erreur")
	}
}
```

Ajouter l'import `"strings"` dans `spotify_test.go`.

Run: `CGO_ENABLED=1 go test ./internal/player/` → Expected : FAIL.

- [ ] **Step 2: Implémenter côté Musique**

Ajouter à `internal/player/applemusic.go` :

```go
// argv : titre de départ (vide = début), aléatoire (1/0), puis les persistent ID dans l'ordre.
const musicPlayListScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	set startID to item 1 of argv
	tell application id "com.apple.Music"
		if not (exists user playlist "Vinyl — Album") then make new user playlist with properties {name:"Vinyl — Album"}
		set p to user playlist "Vinyl — Album"
		delete every track of p
		repeat with i from 3 to count of argv
			try
				duplicate (first track of library playlist 1 whose persistent ID is (item i of argv)) to p
			end try
		end repeat
		set shuffle enabled to ((item 2 of argv) is "1")
		if startID is "" then
			play p
		else
			play (first track of p whose persistent ID is startID)
		end if
	end tell
end run`

// argv : playlist, titre de départ (vide = début), aléatoire (1/0).
const musicPlayInPlaylistScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	tell application id "com.apple.Music"
		set p to (first user playlist whose persistent ID is (item 1 of argv))
		set shuffle enabled to ((item 3 of argv) is "1")
		if (item 2 of argv) is "" then
			play p
		else
			play (first track of p whose persistent ID is (item 2 of argv))
		end if
	end tell
end run`

func flag(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// PlayList remplit la playlist temporaire « Vinyl — Album » avec ids (dans l'ordre) puis joue à partir de startID.
func (m *AppleMusic) PlayList(ctx context.Context, ids []string, startID string, shuffle bool) error {
	out, err := m.run.Run(ctx, musicPlayListScript, append([]string{startID, flag(shuffle)}, ids...)...)
	if err == nil && out == "NOTRUNNING" {
		return ErrNotRunning
	}
	return err
}

// PlayInPlaylist joue une playlist de la bibliothèque à partir d'un titre, la suite continuant dans la playlist.
func (m *AppleMusic) PlayInPlaylist(ctx context.Context, playlistID, startID string, shuffle bool) error {
	out, err := m.run.Run(ctx, musicPlayInPlaylistScript, playlistID, startID, flag(shuffle))
	if err == nil && out == "NOTRUNNING" {
		return ErrNotRunning
	}
	return err
}
```

- [ ] **Step 3: Implémenter côté Spotify**

Ajouter à `internal/player/spotify.go` :

```go
// argv : contexte (album/playlist), titre (vide = début), aléatoire (1/0).
// « play track X in context Y » démarre au titre X dans le contexte Y.
const spotifyPlayInContextScript = `on run argv
	if application id "com.spotify.client" is not running then return "NOTRUNNING"
	tell application id "com.spotify.client"
		set shuffling to ((item 3 of argv) is "1")
		if (item 2 of argv) is "" then
			play track (item 1 of argv)
		else
			try
				play track (item 2 of argv) in context (item 1 of argv)
			on error
				play track (item 1 of argv)
			end try
		end if
	end tell
end run`

func (s *Spotify) PlayInContext(ctx context.Context, contextURI, trackURI string, shuffle bool) error {
	if !strings.HasPrefix(contextURI, "spotify:") {
		return fmt.Errorf("spotify: contexte invalide %q", contextURI)
	}
	shuf := "0"
	if shuffle {
		shuf = "1"
	}
	out, err := s.run.Run(ctx, spotifyPlayInContextScript, contextURI, trackURI, shuf)
	if err == nil && out == "NOTRUNNING" {
		return ErrNotRunning
	}
	return err
}
```

Run: `CGO_ENABLED=1 go test -race ./internal/player/` → Expected : PASS.

Vérifier que les scripts compilent :

```bash
S=$(mktemp -d)
for name in musicPlayListScript musicPlayInPlaylistScript; do
  awk -v n="$name" '$0 ~ "^const "n" = `"{f=1;sub(/^const [A-Za-z]+ = `/,"");print;next} f{ if ($0 ~ /^end run`$/){print "end run"; f=0} else print}' internal/player/applemusic.go > "$S/$name.applescript"
  osacompile -o "$S/$name.scpt" "$S/$name.applescript" && echo "$name OK"
done
```

Expected : deux « OK ». Le script Spotify ne se compile pas sans Spotify installé (−1728) : c'est attendu sur ce Mac.

- [ ] **Step 4: Service**

Ajouter à `internal/app/service.go` :

```go
// PlayFrom lance un album ou une playlist à partir d'un titre (vide = début), avec ou sans aléatoire.
func (s *VinylService) PlayFrom(source, collectionID, trackID string, shuffle bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	defer s.watcher.Wake()
	switch p := s.watcher.Player(source).(type) {
	case *player.AppleMusic:
		if pid, ok := strings.CutPrefix(collectionID, "playlist:"); ok {
			return p.PlayInPlaylist(ctx, pid, trackID, shuffle)
		}
		col, err := s.Tracks(source, collectionID)
		if err != nil {
			return err
		}
		ids := make([]string, len(col.Tracks))
		for i, t := range col.Tracks {
			ids[i] = t.ID
		}
		return p.PlayList(ctx, ids, trackID, shuffle)
	case *player.Spotify:
		return p.PlayInContext(ctx, collectionID, trackID, shuffle)
	}
	return errors.New("source inconnue")
}
```

Run:
```bash
CGO_ENABLED=1 go test . ./internal/... && CGO_ENABLED=1 go vet . ./internal/...
wails3 generate bindings -clean=true -ts -i
```
Expected : PASS.

- [ ] **Step 5: Commit**

```bash
git add -A internal frontend/bindings
git commit -m "feat(lecture): lancer un album ou une playlist à partir d'un titre, avec aléatoire

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 6: Page album / playlist

**Files:**
- Create: `frontend/src/ui/format.ts`, `frontend/src/ui/format.test.ts`, `frontend/src/views/album.ts`
- Modify: `frontend/src/router.ts`, `frontend/src/router.test.ts`, `frontend/src/main.ts`, `frontend/src/api.ts`

**Interfaces:**
- Consumes: service `Tracks`, `PlayFrom` (Tasks 4–5) ; `imageTint`, `paletteFor`, `applyPalette` (`ui/theme.ts`).
- Produces :
  - `formatDuration(ms: number): string`
  - route `"album"` avec param `"<source>/<id>"`
  - `albumParam(source: string, id: string): string`
  - `parseAlbumParam(param: string): { source: string; id: string }`
  - `api.ts` : `Track`, `Collection`

- [ ] **Step 1: Tests**

`frontend/src/ui/format.test.ts` :

```ts
import { expect, test } from "vitest";
import { formatDuration } from "./format";

test("minutes et secondes façon MD Vinyl", () => {
  expect(formatDuration(217000)).toBe("3′37″");
  expect(formatDuration(59000)).toBe("0′59″");
  expect(formatDuration(0)).toBe("0′00″");
});
test("plus d'une heure", () => {
  expect(formatDuration(3_725_000)).toBe("1h02′05″");
});
test("valeurs invalides", () => {
  expect(formatDuration(NaN)).toBe("");
  expect(formatDuration(-5)).toBe("");
});
```

Ajouter à `frontend/src/router.test.ts` (import `albumParam, parseAlbumParam`) :

```ts
test("route album avec source et ID contenant / et séparateur", () => {
  const id = "album:AC/DC\x1fBack in Black";
  const hash = "#/album/" + encodeURIComponent(albumParam("applemusic", id));
  const r = parseRoute(hash);
  expect(r.route).toBe("album");
  expect(parseAlbumParam(r.param)).toEqual({ source: "applemusic", id });
});
```

Run: `cd frontend && npm test` → Expected : FAIL.

- [ ] **Step 2: Implémenter format et route**

`frontend/src/ui/format.ts` :

```ts
// Durée façon MD Vinyl : 3′37″, ou 1h02′05″ au-delà d'une heure.
export function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return "";
  const total = Math.floor(ms / 1000);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = String(total % 60).padStart(2, "0");
  return h > 0 ? `${h}h${String(m).padStart(2, "0")}′${s}″` : `${m}′${s}″`;
}
```

Dans `frontend/src/router.ts` :
- ajouter `"album"` au type `Route` et à `ROUTES` ;
- ajouter :

```ts
export const albumParam = (source: string, id: string) => `${source}/${id}`;
export function parseAlbumParam(param: string): { source: string; id: string } {
  const i = param.indexOf("/");
  return i < 0 ? { source: "applemusic", id: param } : { source: param.slice(0, i), id: param.slice(i + 1) };
}
```

`go("album", albumParam(...))` encode tout le paramètre, et `parseRoute` le décode via `decodeURIComponent(rest.join("/"))`.

Dans `frontend/src/api.ts` :

```ts
export interface Track { id: string; number: number; title: string; artist: string; durationMs: number; loved: boolean; }
export interface Collection { id: string; kind: string; title: string; artist: string; genre: string; year: string; artworkUrl: string; source: string; tracks: Track[] | null; }
```

Run: `cd frontend && npm test` → Expected : PASS.

- [ ] **Step 3: Vue album**

`frontend/src/views/album.ts` :

```ts
// Page album / playlist façon MD Vinyl : pochette avec reflet, infos, Lecture / Lecture aléatoire, liste des titres.
import { api, type Collection } from "../api";
import { go, parseAlbumParam } from "../router";
import { openShareCard } from "../share/sharecard";
import { formatDuration } from "../ui/format";
import { icons } from "../ui/icons";
import { applyPalette, imageTint, paletteFor } from "../ui/theme";
import { escapeHtml } from "./grid";

const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;

export function mount(el: HTMLElement, param: string): () => void {
  const { source, id } = parseAlbumParam(param);
  let alive = true;
  el.innerHTML = `
    <section class="relative h-screen overflow-hidden">
      <button id="al-back" class="glass round-btn vy-drop fixed z-30" style="left:92px;top:28px" aria-label="Retour">${icons.back}</button>
      <button id="al-share" class="glass round-btn vy-drop fixed z-30" style="right:51px;top:28px" aria-label="Partager">${icons.share}</button>
      <h1 id="al-bar" class="pointer-events-none fixed left-0 z-20 text-center text-[24px] font-semibold opacity-0 transition-opacity" style="right:130px;top:36px;color:var(--ink)"></h1>
      <div id="al-body" class="absolute inset-0 pt-[100px] text-[15px]" style="right:130px;color:var(--ink-soft)">Chargement…</div>
    </section>`;
  const $ = (q: string) => el.querySelector<HTMLElement>("#" + q)!;
  $("al-back").onclick = () => history.length > 1 ? history.back() : go("bibliotheque", "albums");

  const play = async (trackID: string, shuffle: boolean) => {
    try {
      await api.PlayFrom(source, id, trackID, shuffle);
      go("platine");
    } catch (e) {
      $("al-err").textContent = String(e);
    }
  };

  const render = (c: Collection) => {
    const tracks = c.tracks ?? [];
    const meta = [c.genre, c.year].filter(Boolean).join(" · ");
    $("al-bar").textContent = c.title;
    $("al-body").innerHTML = `
      <div class="flex h-full gap-14 px-[92px]">
        <div class="vy-fly shrink-0" style="width:min(42vw,520px)">
          <div class="aspect-square w-full bg-cover bg-center shadow-2xl" style="background-color:#2a2220;${c.artworkUrl ? `background-image:${escapeHtml(cssUrl(c.artworkUrl))}` : ""};-webkit-box-reflect:below 4px linear-gradient(transparent 70%, rgba(0,0,0,.25))"></div>
        </div>
        <div id="al-scroll" class="shelf-scroll flex-1 overflow-y-auto pb-24" style="color:var(--ink)">
          <header id="al-head" class="vy-slide flex flex-col items-center gap-3 pb-10 pt-8 text-center">
            <h2 class="text-[40px] font-black leading-tight">${escapeHtml(c.title)}</h2>
            <p class="text-[26px] font-semibold opacity-90">${escapeHtml(c.artist)}</p>
            ${meta ? `<p class="text-[19px] opacity-75">${escapeHtml(meta)}</p>` : ""}
            <div class="mt-6 flex gap-5">
              <button id="al-play" class="pill border-2 px-10 py-3 text-[18px]" style="border-color:var(--ink)">Lecture</button>
              <button id="al-shuffle" class="pill border-2 px-10 py-3 text-[18px]" style="border-color:var(--ink)">Lecture aléatoire</button>
            </div>
            <p id="al-err" class="text-[14px] text-red-300"></p>
          </header>
          <ol class="flex flex-col">${tracks.map((t, i) => `
            <li><button data-i="${i}" class="flex w-full items-center gap-5 border-b py-5 text-left text-[20px] hover:opacity-80" style="border-color:color-mix(in srgb, var(--ink) 15%, transparent)">
              <span class="w-6 text-right">${t.loved ? "♥" : ""}</span>
              <span class="w-8 tabular-nums opacity-80">${String(t.number || i + 1).padStart(2, "0")}</span>
              <span class="flex-1 truncate font-semibold">${escapeHtml(t.title)}</span>
              <span class="tabular-nums opacity-80">${formatDuration(t.durationMs)}</span>
            </button></li>`).join("")}</ol>
        </div>
      </div>`;
    $("al-play").onclick = () => play("", false);
    $("al-shuffle").onclick = () => play("", true);
    el.querySelectorAll<HTMLElement>("[data-i]").forEach((b) => (b.onclick = () => play(tracks[Number(b.dataset.i)]!.id, false)));
    // En défilant, l'en-tête laisse place à une barre de titre centrée.
    const sc = $("al-scroll");
    sc.addEventListener("scroll", () => $("al-bar").classList.toggle("opacity-0", sc.scrollTop < $("al-head").offsetHeight - 40), { passive: true });
    $("al-share").onclick = () => openShareCard({ title: c.title, subtitle: c.artist, artworkUrl: c.artworkUrl });
    imageTint(c.artworkUrl).then((tint) => { if (alive && tint) applyPalette(paletteFor(tint)); });
  };

  const load = () => {
    $("al-body").textContent = "Chargement…";
    api.Tracks(source, id).then((c) => { if (alive) render(c); }).catch((e) => {
      if (!alive) return;
      $("al-body").innerHTML = `<div class="px-[92px] pt-10">${escapeHtml(String(e))} <button id="al-retry" class="underline">Réessayer</button></div>`;
      $("al-retry").onclick = load;
    });
  };
  load();
  return () => { alive = false; };
}
```

Le fond global (calques `bg-a`/`bg-b` du shell) est piloté par le morceau en cours. `applyPalette` ne change que les variables de couleur du texte et du verre. Pour teinter aussi le fond de la page, ajouter dans `render`, après `applyPalette` :

```ts
      el.querySelector<HTMLElement>("section")!.style.background = paletteFor(tint).bg;
```

Dans `frontend/src/main.ts` : `import * as album from "./views/album";` et ajouter `album` à l'objet `views`.

Run: `cd frontend && npm test && npx tsc --noEmit` → Expected : PASS.

- [ ] **Step 4: Commit**

```bash
git add frontend/src
git commit -m "feat(album): page album/playlist avec titres, durées, cœurs et lecture aléatoire

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 7: Modes de lecture et titres à venir (backend)

**Files:**
- Modify: `internal/player/player.go`, `internal/player/applemusic.go`, `internal/player/spotify.go`, `internal/catalog/spotifyweb.go`, `internal/app/service.go`
- Test: `internal/player/applemusic_test.go`, `internal/player/spotify_test.go`, `internal/catalog/spotifyweb_test.go`

**Interfaces:**
- Produces :
  - `player.RepeatMode` (`RepeatOff = "off"`, `RepeatAll = "all"`, `RepeatOne = "one"`)
  - `player.Modes` interface { `PlaybackModes(ctx) (bool, RepeatMode, error)` ; `SetShuffle(ctx, bool) error` ; `SetRepeat(ctx, RepeatMode) error` }
  - `(*AppleMusic).Upcoming(ctx, limit int, wrap bool) ([]Track, error)`
  - `(*catalog.SpotifyWeb).Queue(ctx) ([]player.Track, error)`
  - `app.QueueState{Shuffle bool; Repeat string; Upcoming []player.Track; Note string}` (tags `shuffle, repeat, upcoming, note`)
  - service `Queue() (QueueState, error)`, `SetShuffle(on bool) error`, `SetRepeat(mode string) error`

- [ ] **Step 1: Tests**

Ajouter à `internal/player/applemusic_test.go` :

```go
func TestMusicPlaybackModes(t *testing.T) {
	r := &fakeRunner{outputs: []string{join("true", "all")}}
	sh, rep, err := NewAppleMusic(r, t.TempDir()).PlaybackModes(context.Background())
	if err != nil || !sh || rep != RepeatAll {
		t.Errorf("got %v %v %v", sh, rep, err)
	}
	r = &fakeRunner{outputs: []string{join("false", "bizarre")}}
	_, rep, _ = NewAppleMusic(r, t.TempDir()).PlaybackModes(context.Background())
	if rep != RepeatOff {
		t.Errorf("valeur inconnue → off, got %v", rep)
	}
}

func TestMusicSetRepeatAndShuffleArgs(t *testing.T) {
	r := &fakeRunner{}
	m := NewAppleMusic(r, t.TempDir())
	m.SetRepeat(context.Background(), RepeatOne)
	m.SetShuffle(context.Background(), true)
	if r.calls[0].args[0] != "one" || r.calls[1].args[0] != "1" {
		t.Errorf("args = %+v", r.calls)
	}
	if err := m.SetRepeat(context.Background(), "jamais"); err == nil {
		t.Error("mode inconnu → erreur")
	}
}

func TestMusicUpcomingAtEndOfPlaylist(t *testing.T) {
	r := &fakeRunner{outputs: []string{"5" + Sep + "5" + "\x1e"}} // index 5 sur 5 : rien après
	ts, err := NewAppleMusic(r, t.TempDir()).Upcoming(context.Background(), 30, false)
	if err != nil || len(ts) != 0 {
		t.Errorf("got %v %v", ts, err)
	}
	if strings.Join(r.calls[0].args, "|") != "30|0" {
		t.Errorf("args = %q", r.calls[0].args)
	}
}

func TestMusicUpcomingParsesTracks(t *testing.T) {
	out := "2" + Sep + "4" + "\x1e" + join("X", "Trois", "A", "200") + "\x1e" + join("Y", "Quatre", "B", "180,5") + "\x1e"
	ts, _ := NewAppleMusic(&fakeRunner{outputs: []string{out}}, t.TempDir()).Upcoming(context.Background(), 30, true)
	if len(ts) != 2 || ts[1].Title != "Quatre" || ts[1].DurationMs != 180500 {
		t.Errorf("got %+v", ts)
	}
}

func TestMusicUpcomingNoCurrentPlaylist(t *testing.T) {
	ts, err := NewAppleMusic(&fakeRunner{outputs: []string{"NONE"}}, t.TempDir()).Upcoming(context.Background(), 30, false)
	if err != nil || ts != nil {
		t.Errorf("radio/flux : liste vide sans erreur, got %v %v", ts, err)
	}
}
```

Ajouter à `internal/player/spotify_test.go` :

```go
func TestSpotifyModes(t *testing.T) {
	r := &fakeRunner{outputs: []string{join("false", "true")}}
	s := NewSpotify(r)
	sh, rep, _ := s.PlaybackModes(context.Background())
	if sh || rep != RepeatAll {
		t.Errorf("got %v %v", sh, rep)
	}
	s.SetRepeat(context.Background(), RepeatOne)
	if r.calls[1].args[0] != "1" {
		t.Errorf("« un seul » n'existe pas sur Spotify : traité comme « tout », args = %q", r.calls[1].args)
	}
}
```

Ajouter à `internal/catalog/spotifyweb_test.go` :

```go
func TestSpotifyQueue(t *testing.T) {
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/player/queue" {
			http.Error(w, r.URL.Path, 404)
			return
		}
		w.Write([]byte(`{"currently_playing":{"uri":"spotify:track:0"},"queue":[{"uri":"spotify:track:1","name":"Heart Flutters","duration_ms":1000,"artists":[{"name":"Egofear"}]}]}`))
	})
	q, err := c.Queue(context.Background())
	if err != nil || len(q) != 1 || q[0].Title != "Heart Flutters" || q[0].Artist != "Egofear" {
		t.Errorf("got %+v %v", q, err)
	}
}
```

Run: `CGO_ENABLED=1 go test ./internal/player/ ./internal/catalog/` → Expected : FAIL.

- [ ] **Step 2: Implémenter**

Ajouter à `internal/player/player.go` :

```go
type RepeatMode string

const (
	RepeatOff RepeatMode = "off"
	RepeatAll RepeatMode = "all"
	RepeatOne RepeatMode = "one"
)

// Modes est implémentée par les lecteurs qui savent lire et régler l'aléatoire et la répétition.
type Modes interface {
	PlaybackModes(ctx context.Context) (shuffle bool, repeat RepeatMode, err error)
	SetShuffle(ctx context.Context, on bool) error
	SetRepeat(ctx context.Context, mode RepeatMode) error
}

func parseRepeat(s string) RepeatMode {
	switch RepeatMode(strings.TrimSpace(s)) {
	case RepeatAll:
		return RepeatAll
	case RepeatOne:
		return RepeatOne
	}
	return RepeatOff
}
```

(Ajouter l'import `"strings"` dans `player.go`.)

Ajouter à `internal/player/applemusic.go` :

```go
const musicModesScript = musicGuard + `tell application id "com.apple.Music"
	return ((shuffle enabled) as string) & (character id 31) & ((song repeat) as string)
end tell`

const musicSetShuffleScript = `on run argv
	tell application id "com.apple.Music" to set shuffle enabled to ((item 1 of argv) is "1")
end run`

const musicSetRepeatScript = `on run argv
	tell application id "com.apple.Music"
		if (item 1 of argv) is "one" then
			set song repeat to one
		else if (item 1 of argv) is "all" then
			set song repeat to all
		else
			set song repeat to off
		end if
	end tell
end run`

// argv : limite, reprise au début (1/0). Sortie : "index␟total␞" puis un enregistrement par titre.
const musicUpcomingScript = `on run argv
	if application id "com.apple.Music" is not running then return "NOTRUNNING"
	set us to character id 31
	set rs to character id 30
	set lim to (item 1 of argv) as integer
	tell application id "com.apple.Music"
		try
			set cp to current playlist
			set idx to index of current track
		on error
			return "NONE"
		end try
		set n to count of tracks of cp
		set out to (idx as string) & us & (n as string) & rs
		set k to 0
		set i to idx + 1
		repeat while k < lim and k < n - 1
			if i > n then
				if (item 2 of argv) is not "1" then exit repeat
				set i to 1
			end if
			set t to track i of cp
			set out to out & (persistent ID of t) & us & (name of t) & us & (artist of t) & us & ((duration of t) as string) & rs
			set k to k + 1
			set i to i + 1
		end repeat
		return out
	end tell
end run`

func (m *AppleMusic) PlaybackModes(ctx context.Context) (bool, RepeatMode, error) {
	out, err := m.run.Run(ctx, musicModesScript)
	if err != nil {
		return false, RepeatOff, err
	}
	if out == "NOTRUNNING" {
		return false, RepeatOff, ErrNotRunning
	}
	sh, rep, _ := strings.Cut(out, Sep)
	return sh == "true", parseRepeat(rep), nil
}

func (m *AppleMusic) SetShuffle(ctx context.Context, on bool) error {
	_, err := m.run.Run(ctx, musicSetShuffleScript, flag(on))
	return err
}

func (m *AppleMusic) SetRepeat(ctx context.Context, mode RepeatMode) error {
	if mode != RepeatOff && mode != RepeatAll && mode != RepeatOne {
		return fmt.Errorf("applemusic: mode de répétition inconnu %q", mode)
	}
	_, err := m.run.Run(ctx, musicSetRepeatScript, string(mode))
	return err
}

// Upcoming renvoie les titres qui suivent dans la playlist en cours (nil si aucune : radio, flux).
func (m *AppleMusic) Upcoming(ctx context.Context, limit int, wrap bool) ([]Track, error) {
	out, err := m.run.Run(ctx, musicUpcomingScript, strconv.Itoa(limit), flag(wrap))
	if err != nil {
		return nil, err
	}
	switch out {
	case "NOTRUNNING":
		return nil, ErrNotRunning
	case "NONE", "":
		return nil, nil
	}
	recs := strings.Split(out, "\x1e")
	tracks := []Track{}
	for _, r := range recs[1:] {
		f := strings.Split(r, Sep)
		if len(f) == 4 {
			tracks = append(tracks, Track{ID: f[0], Title: f[1], Artist: f[2], DurationMs: parseSeconds(f[3]), Number: len(tracks) + 1})
		}
	}
	return tracks, nil
}
```

Ajouter à `internal/player/spotify.go` :

```go
const spotifyModesScript = spotifyGuard + `tell application id "com.spotify.client"
	return ((shuffling) as string) & (character id 31) & ((repeating) as string)
end tell`

func (s *Spotify) PlaybackModes(ctx context.Context) (bool, RepeatMode, error) {
	out, err := s.run.Run(ctx, spotifyModesScript)
	if err != nil {
		return false, RepeatOff, err
	}
	if out == "NOTRUNNING" {
		return false, RepeatOff, ErrNotRunning
	}
	sh, rep, _ := strings.Cut(out, Sep)
	mode := RepeatOff
	if rep == "true" {
		mode = RepeatAll
	}
	return sh == "true", mode, nil
}

func (s *Spotify) SetShuffle(ctx context.Context, on bool) error {
	v := "0"
	if on {
		v = "1"
	}
	_, err := s.run.Run(ctx, `on run argv
	tell application id "com.spotify.client" to set shuffling to ((item 1 of argv) is "1")
end run`, v)
	return err
}

// SetRepeat : Spotify ne connaît que oui/non ; « un seul » est traité comme « tout ».
func (s *Spotify) SetRepeat(ctx context.Context, mode RepeatMode) error {
	v := "0"
	if mode == RepeatAll || mode == RepeatOne {
		v = "1"
	}
	_, err := s.run.Run(ctx, `on run argv
	tell application id "com.spotify.client" to set repeating to ((item 1 of argv) is "1")
end run`, v)
	return err
}
```

Ajouter à `internal/catalog/spotifyweb.go` :

```go
// Queue renvoie les titres de la file Spotify (30 au plus).
func (c *SpotifyWeb) Queue(ctx context.Context) ([]player.Track, error) {
	var q struct {
		Queue []spFullTrack `json:"queue"`
	}
	if err := c.get(ctx, "/me/player/queue", nil, &q); err != nil {
		return nil, err
	}
	out := []player.Track{}
	for i, t := range q.Queue {
		if i == 30 {
			break
		}
		out = append(out, toTrack(t, i+1))
	}
	return out, nil
}
```

Run: `CGO_ENABLED=1 go test -race ./internal/player/ ./internal/catalog/` → Expected : PASS.

- [ ] **Step 3: Service**

Ajouter à `internal/app/service.go` :

```go
type QueueState struct {
	Shuffle  bool           `json:"shuffle"`
	Repeat   string         `json:"repeat"`
	Upcoming []player.Track `json:"upcoming"`
	Note     string         `json:"note"`
}

func (s *VinylService) modes() (player.Modes, error) {
	p := s.watcher.Active()
	if p == nil {
		return nil, errNoPlayer
	}
	m, ok := p.(player.Modes)
	if !ok {
		return nil, errors.New("modes de lecture non pris en charge")
	}
	return m, nil
}

// Queue renvoie l'aléatoire, la répétition et les titres à venir du lecteur actif.
func (s *VinylService) Queue() (QueueState, error) {
	m, err := s.modes()
	if err != nil {
		return QueueState{Upcoming: []player.Track{}}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	sh, rep, err := m.PlaybackModes(ctx)
	if err != nil {
		return QueueState{Upcoming: []player.Track{}}, err
	}
	q := QueueState{Shuffle: sh, Repeat: string(rep), Upcoming: []player.Track{}}
	switch p := m.(type) {
	case *player.AppleMusic:
		if sh {
			q.Note = "Ordre aléatoire"
			break
		}
		if ts, err := p.Upcoming(ctx, 30, rep == player.RepeatAll); err == nil && ts != nil {
			q.Upcoming = ts
		}
	case *player.Spotify:
		web, ok := s.catalogs["spotify"].(*catalog.SpotifyWeb)
		if !ok || s.spotifyAuth == nil || !s.spotifyAuth.LoggedIn() {
			q.Note = "Connecte Spotify pour voir la suite"
			break
		}
		if ts, err := web.Queue(ctx); err == nil {
			q.Upcoming = ts
		} else {
			q.Note = "File Spotify indisponible"
		}
	}
	return q, nil
}

func (s *VinylService) SetShuffle(on bool) error {
	m, err := s.modes()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	return m.SetShuffle(ctx, on)
}

func (s *VinylService) SetRepeat(mode string) error {
	m, err := s.modes()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	return m.SetRepeat(ctx, player.RepeatMode(mode))
}
```

Vérifier que les nouveaux scripts Musique compilent :

```bash
S=$(mktemp -d)
for name in musicSetShuffleScript musicSetRepeatScript musicUpcomingScript; do
  awk -v n="$name" '$0 ~ "^const "n" = `"{f=1;sub(/^const [A-Za-z]+ = `/,"");print;next} f{ if ($0 ~ /^end run`$/){print "end run"; f=0} else print}' internal/player/applemusic.go > "$S/$name.applescript"
  osacompile -o "$S/$name.scpt" "$S/$name.applescript" && echo "$name OK"
done
VINYL_LIVE=1 CGO_ENABLED=1 go test ./internal/player/ -run Live -v -count=1
```

Expected : trois « OK ». Pour `musicModesScript` (qui commence par `musicGuard`), ajouter une ligne au test live `TestLiveNowPlaying` :

```go
	if m, ok := Player(NewAppleMusic(OSAScript{}, t.TempDir())).(Modes); ok {
		_, _, err := m.PlaybackModes(context.Background())
		if err != nil && err != ErrNotRunning && err != ErrNotAuthorized {
			t.Errorf("modes : %v", err)
		}
	}
```

Puis :

```bash
CGO_ENABLED=1 go test . ./internal/... && CGO_ENABLED=1 go vet . ./internal/...
wails3 generate bindings -clean=true -ts -i
```

Expected : PASS.

- [ ] **Step 4: Commit**

```bash
git add -A internal frontend/bindings
git commit -m "feat(file): aléatoire, répétition et titres à venir (Musique et Spotify)

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 8: Étagère coverflow

**Files:**
- Create: `frontend/src/views/coverflow.ts`, `frontend/src/views/coverflow.test.ts`
- Modify: `frontend/src/views/bibliotheque.ts`, `frontend/src/styles.css`, `frontend/src/router.ts`, `frontend/src/router.test.ts`

**Interfaces:**
- Consumes: `go("album", albumParam(source, id))` (Task 6).
- Produces :
  - `slotTransform(i: number, focus: number): { x: number; rotateY: number; z: number; zIndex: number }`
  - `visibleRange(focus: number, count: number, radius?: number): [number, number]`
  - `class WheelStepper { constructor(step?: number); push(delta: number): number }` (renvoie le nombre de crans, signé)

- [ ] **Step 1: Tests**

`frontend/src/views/coverflow.test.ts` :

```ts
import { expect, test } from "vitest";
import { slotTransform, visibleRange, WheelStepper } from "./coverflow";

test("élément sélectionné : de face, avancé, au-dessus", () => {
  expect(slotTransform(5, 5)).toEqual({ x: 0, rotateY: 0, z: 120, zIndex: 1000 });
});
test("à gauche : +70°, à droite : −70°, espacés de 64 px après le grand saut", () => {
  const l1 = slotTransform(4, 5), l2 = slotTransform(3, 5), r1 = slotTransform(6, 5);
  expect(l1.rotateY).toBe(70);
  expect(r1.rotateY).toBe(-70);
  expect(l1.x).toBeLessThan(0);
  expect(r1.x).toBeGreaterThan(0);
  expect(l1.x - l2.x).toBe(64);
  expect(l1.zIndex).toBeGreaterThan(l2.zIndex);
});
test("fenêtre ±12 bornée", () => {
  expect(visibleRange(0, 100)).toEqual([0, 12]);
  expect(visibleRange(50, 100)).toEqual([38, 62]);
  expect(visibleRange(98, 100)).toEqual([86, 99]);
  expect(visibleRange(0, 0)).toEqual([0, -1]);
});
test("molette : un cran tous les 60 px, reste conservé", () => {
  const w = new WheelStepper();
  expect(w.push(40)).toBe(0);
  expect(w.push(30)).toBe(1);
  expect(w.push(-130)).toBe(-2);
});
```

Ajouter à `frontend/src/router.test.ts` :

```ts
test("l'étagère ne propose plus que Albums et Playlists ; anciens onglets → albums", () => {
  expect(parseRoute("#/bibliotheque/playlists")).toEqual({ route: "bibliotheque", param: "playlists" });
  expect(parseRoute("#/bibliotheque/likes")).toEqual({ route: "bibliotheque", param: "likes" });
});
```

(« likes » et « artistes » restent des routes valides de l'étagère, accessibles par le menu. Seule la commande segmentée se limite à deux onglets.)

Run: `cd frontend && npm test` → Expected : FAIL (module coverflow absent).

- [ ] **Step 2: Implémenter la géométrie**

`frontend/src/views/coverflow.ts` :

```ts
// Coverflow façon MD Vinyl : sélection de face au centre, voisins inclinés à ±70° montrant leur tranche.
export const SPINE_GAP = 64;
const FIRST_GAP = 230; // écart entre la sélection et le premier voisin
export const RADIUS = 12;

export function slotTransform(i: number, focus: number): { x: number; rotateY: number; z: number; zIndex: number } {
  const d = i - focus;
  if (d === 0) return { x: 0, rotateY: 0, z: 120, zIndex: 1000 };
  const side = Math.sign(d);
  const x = side * (FIRST_GAP + (Math.abs(d) - 1) * SPINE_GAP);
  return { x, rotateY: -side * 70, z: 0, zIndex: 1000 - Math.abs(d) };
}

export function visibleRange(focus: number, count: number, radius = RADIUS): [number, number] {
  if (count <= 0) return [0, -1];
  return [Math.max(0, focus - radius), Math.min(count - 1, focus + radius)];
}

export class WheelStepper {
  private acc = 0;
  constructor(private step = 60) {}
  push(delta: number): number {
    this.acc += delta;
    const n = Math.trunc(this.acc / this.step);
    this.acc -= n * this.step;
    return n;
  }
}
```

Run: `cd frontend && npm test` → Expected : PASS.

- [ ] **Step 3: Brancher dans `bibliotheque.ts`**

Remplacer le mode « side » (rangée `.vy-row` défilante) par le coverflow, en gardant le mode pile (« flat »).

1. **En-tête**
   - Les boutons `sh-back` et `sh-mode` deviennent un seul bouton menu `sh-menu` (icône `icons.stack`) en `left:92px;top:28px`.
   - Il ouvre une liste en verre (`id="sh-menu-pop"`, classe `glass fixed z-30 hidden w-64 rounded-[30px] p-2`, `left:92px;top:88px`) contenant :
     - « Platine » → `go("platine")` ;
     - « Artistes » → `go("bibliotheque","artistes")` ;
     - « Titres likés » → `go("bibliotheque","likes")` ;
     - « Vue pile » / « Vue étagère », qui bascule `mode`.
   - La commande segmentée n'affiche que `["albums", "playlists"]`. Si l'onglet courant est « artistes » ou « likes », aucune pastille n'est active et un titre `<h2>` en verre affiche « Artistes » ou « Titres likés ».

2. **Rendu en mode `side`** : remplacer le contenu de `area` par une scène fixe :

```ts
const stageHtml = `<div id="cf" class="relative h-full w-full" style="perspective:1300px;perspective-origin:50% 40%">
  <div id="cf-row" class="absolute left-1/2 top-[44%]" style="transform-style:preserve-3d"></div></div>`;
```

et une fonction `layout()` :

```ts
let focus = 0;
const SIZE = 300; // boîtier carré
const layout = () => {
  const row = area.querySelector<HTMLElement>("#cf-row");
  if (!row) return;
  const shown = filterItems(items, query);
  focus = Math.min(focus, Math.max(0, shown.length - 1));
  const [a, b] = visibleRange(focus, shown.length);
  const keep = new Set<string>();
  for (let i = a; i <= b; i++) {
    const it = shown[i]!;
    keep.add(it.id);
    let slot = row.querySelector<HTMLElement>(`[data-id="${CSS.escape(it.id)}"]`);
    if (!slot) {
      row.insertAdjacentHTML("beforeend", itemHtml(it, i, false));
      slot = row.lastElementChild as HTMLElement;
    }
    const t = slotTransform(i, focus);
    slot.style.transform = `translate(-50%,-50%) translateX(${t.x}px) translateZ(${t.z}px) rotateY(${t.rotateY}deg)`;
    slot.style.zIndex = String(t.zIndex);
    slot.dataset.index = String(i);
  }
  row.querySelectorAll<HTMLElement>("[data-id]").forEach((n) => { if (!keep.has(n.dataset.id!)) n.remove(); });
  if (b >= shown.length - 6) maybeLoad();
};
```

3. **HTML d'un élément en mode `side`** : un boîtier `button.vy-cf` de `SIZE×SIZE` contenant :
   - la face (`.vy-face`, pochette) ;
   - la tranche (`.vy-spine`, déjà stylée) placée sur le côté visible : à gauche pour les éléments de droite, à droite pour ceux de gauche.

   Pour simplifier : la tranche est toujours rendue sur la gauche (`left:-34px`). Pour les éléments à gauche de la sélection, `layout()` ajoute la classe `cf-left`, qui la déplace à droite via CSS.

   Remplacer dans `styles.css` les règles `.vy-row`, `.vy-slot`, `.vy-box` et les règles `:hover` associées par :

```css
/* Coverflow : boîtiers carrés, transition douce, tranche du côté visible, reflet au sol. */
.vy-cf { position: absolute; left: 0; top: 0; width: 300px; height: 300px; border: 0; padding: 0; background: none; cursor: pointer; transform-style: preserve-3d; transition: transform .55s cubic-bezier(.25,.8,.25,1); -webkit-box-reflect: below 6px linear-gradient(transparent 62%, rgba(0,0,0,.22)); }
.vy-cf .vy-face { position: absolute; inset: 0; }
.vy-cf .vy-spine { left: -34px; }
.vy-cf.cf-left .vy-spine { left: auto; right: -34px; transform-origin: 0 50%; transform: rotateY(90deg); }
```

   `.vy-face` et `.vy-spine` existants restent (couleurs, texte vertical).

4. **Interactions** (mode `side`) :
   - `wheel` sur `#cf` : `const n = stepper.push(Math.abs(e.deltaX) > Math.abs(e.deltaY) ? e.deltaX : e.deltaY); if (n) { focus = clamp(focus + n); layout(); } e.preventDefault();` avec `const stepper = new WheelStepper();` ;
   - `keydown` sur `window` (retiré au nettoyage) : `ArrowLeft` / `ArrowRight` déplacent `focus`, `Enter` ouvre la sélection ;
   - clic sur un `.vy-cf` : si `Number(dataset.index) !== focus`, `focus = index; layout()` ; sinon `open(it)` ;
   - `open(it)` :
     - `it.kind === "artist"` → `go("artiste", it.title)` ;
     - `it.kind === "track"` → `api.PlayItem(source, it.id)` puis `go("platine")` (titres likés) ;
     - sinon → `go("album", albumParam(source, it.id))`.
   - Le mode pile garde son comportement, mais ses clics passent aussi par `open(it)`, pour ouvrir la page album au lieu de lancer la lecture.

5. Après chaque `load()` réussi en mode `side`, appeler `layout()` au lieu de reconstruire tout le HTML. Ne pas réinitialiser `focus`.

Imports à ajouter : `slotTransform`, `visibleRange`, `WheelStepper` depuis `./coverflow` ; `albumParam` depuis `../router`.

Run: `cd frontend && npm test && npx tsc --noEmit` → Expected : PASS.

- [ ] **Step 4: Commit**

```bash
git add frontend/src
git commit -m "feat(étagère): coverflow 3D avec tranches nommées, molette et clavier ; ouvre la page album

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 9: Panneau « File d'attente »

**Files:**
- Create: `frontend/src/shell/queue.ts`, `frontend/src/shell/repeat.ts`, `frontend/src/shell/repeat.test.ts`
- Modify: `frontend/src/shell/shell.ts`, `frontend/src/ui/icons.ts`, `frontend/src/api.ts`

**Interfaces:**
- Consumes: service `Queue`, `SetShuffle`, `SetRepeat` (Task 7) ; `formatDuration` (Task 6).
- Produces :
  - `QueueState` dans `api.ts`
  - `nextRepeat(mode: string): "off" | "all" | "one"` (dans `shell/repeat.ts`)
  - `mountQueuePanel(root: HTMLElement): { toggle(force?: boolean): void; refresh(): void; isOpen(): boolean }`

- [ ] **Step 1: Test du cycle de répétition**

Créer `frontend/src/shell/repeat.test.ts` (module pur, sans import des bindings Wails) :

```ts
import { expect, test } from "vitest";
import { nextRepeat } from "./repeat";

test("répéter : aucun → tout → un seul → aucun", () => {
  expect(nextRepeat("off")).toBe("all");
  expect(nextRepeat("all")).toBe("one");
  expect(nextRepeat("one")).toBe("off");
  expect(nextRepeat("??")).toBe("all");
});
```

Run: `cd frontend && npm test` → Expected : FAIL.

Puis créer `frontend/src/shell/repeat.ts` :

```ts
// Cycle du bouton « Répéter » : aucun → tout → un seul → aucun.
export function nextRepeat(mode: string): "off" | "all" | "one" {
  return mode === "all" ? "one" : mode === "one" ? "off" : "all";
}
```

Run: `cd frontend && npm test` → Expected : PASS.

- [ ] **Step 2: Icônes et types**

Ajouter à l'objet `icons` de `frontend/src/ui/icons.ts` :

```ts
  shuffle: svg(22, '<path d="M3 7h3.5c2.6 0 4 1.2 5.5 5s2.9 5 5.5 5H21"/><path d="M3 17h3.5c1.4 0 2.5-.4 3.4-1.3"/><path d="M14.1 8.3C15 7.4 16.1 7 17.5 7H21"/><path d="M18 4l3 3-3 3"/><path d="M18 14l3 3-3 3"/>'),
  repeat: svg(22, '<path d="M4 11V9a3 3 0 0 1 3-3h13"/><path d="M17 3l3 3-3 3"/><path d="M20 13v2a3 3 0 0 1-3 3H4"/><path d="M7 21l-3-3 3-3"/>'),
  repeatOne: svg(22, '<path d="M4 11V9a3 3 0 0 1 3-3h13"/><path d="M17 3l3 3-3 3"/><path d="M20 13v2a3 3 0 0 1-3 3H4"/><path d="M7 21l-3-3 3-3"/><path d="M11.5 10.5l1.5-1v5"/>'),
```

Ajouter à `frontend/src/api.ts` :

```ts
export interface QueueState { shuffle: boolean; repeat: string; upcoming: Track[] | null; note: string; }
```

- [ ] **Step 3: Panneau**

`frontend/src/shell/queue.ts` :

```ts
// Panneau « File d'attente » : pochette inclinée, morceau en cours, titres suivants, aléatoire / répéter.
import { api, onSnapshot, type QueueState, type Snapshot } from "../api";
import { formatDuration } from "../ui/format";
import { icons } from "../ui/icons";
import { escapeHtml } from "../views/grid";
import { nextRepeat } from "./repeat";

const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;

export function mountQueuePanel(root: HTMLElement): { toggle(force?: boolean): void; refresh(): void; isOpen(): boolean } {
  const panel = document.createElement("aside");
  panel.className = "fixed z-40 hidden flex-col gap-4 overflow-hidden rounded-[34px] p-7";
  panel.style.cssText = "right:130px;top:28px;bottom:28px;width:380px;background:rgba(28,20,18,.72);-webkit-backdrop-filter:blur(28px) saturate(1.4);backdrop-filter:blur(28px) saturate(1.4);color:#f6ece6;box-shadow:0 24px 60px rgba(0,0,0,.45)";
  panel.setAttribute("aria-label", "File d'attente");
  panel.innerHTML = `
    <div id="q-cover" class="mx-auto h-40 w-40 shrink-0 bg-cover bg-center shadow-2xl" style="transform:rotate(-4deg);background-color:#2a2220"></div>
    <h2 class="text-center text-[17px] font-bold">File d'attente</h2>
    <div id="q-now" class="border-b border-white/15 pb-3"></div>
    <ol id="q-list" class="shelf-scroll flex-1 overflow-y-auto"></ol>
    <p id="q-note" class="text-center text-[13px] opacity-70"></p>
    <div class="flex justify-center gap-5">
      <button id="q-shuffle" class="round-btn" aria-label="Lecture aléatoire">${icons.shuffle}</button>
      <button id="q-repeat" class="round-btn" aria-label="Répéter">${icons.repeat}</button>
    </div>`;
  root.appendChild(panel);
  const $ = (id: string) => panel.querySelector<HTMLElement>("#" + id)!;
  let state: QueueState | null = null;
  let lastTrack = "";

  const paintToggle = (btn: HTMLElement, on: boolean) => {
    btn.style.background = on ? "#f6ece6" : "rgba(255,255,255,.12)";
    btn.style.color = on ? "#1c1412" : "#f6ece6";
    btn.setAttribute("aria-pressed", String(on));
  };

  const render = (q: QueueState) => {
    state = q;
    const up = q.upcoming ?? [];
    $("q-list").innerHTML = up.map((t) => `<li class="flex items-baseline gap-3 py-2"><span class="flex-1 truncate"><b class="font-semibold">${escapeHtml(t.title)}</b><br><span class="text-[13px] opacity-70">${escapeHtml(t.artist)}</span></span><span class="text-[13px] tabular-nums opacity-60">${formatDuration(t.durationMs)}</span></li>`).join("");
    $("q-note").textContent = q.note || (up.length === 0 ? "Rien à suivre" : "");
    paintToggle($("q-shuffle"), q.shuffle);
    $("q-repeat").innerHTML = q.repeat === "one" ? icons.repeatOne : icons.repeat;
    paintToggle($("q-repeat"), q.repeat !== "off");
  };

  const refresh = () => {
    if (panel.classList.contains("hidden")) return;
    api.Queue().then(render).catch((e) => { $("q-note").textContent = String(e); });
  };

  const showNow = (s: Snapshot) => {
    const np = s.nowPlaying;
    $("q-cover").style.backgroundImage = np.artworkUrl ? cssUrl(np.artworkUrl) : "";
    $("q-now").innerHTML = s.problem ? "" : `<b>${escapeHtml(np.title)}</b><br><span class="text-[13px] opacity-70">${escapeHtml(np.artist)}</span>`;
    if (np.trackId !== lastTrack) { lastTrack = np.trackId; refresh(); }
  };

  $("q-shuffle").onclick = async () => { await api.SetShuffle(!(state?.shuffle ?? false)).catch(() => {}); refresh(); };
  $("q-repeat").onclick = async () => { await api.SetRepeat(nextRepeat(state?.repeat ?? "off")).catch(() => {}); refresh(); };

  const toggle = (force?: boolean) => {
    const open = force ?? panel.classList.contains("hidden");
    panel.classList.toggle("hidden", !open);
    panel.classList.toggle("flex", open);
    if (open) { panel.classList.add("vy-slide"); refresh(); }
  };
  document.addEventListener("keydown", (e) => { if (e.key === "Escape") toggle(false); });
  document.addEventListener("click", (e) => {
    const t = e.target as HTMLElement;
    if (!panel.classList.contains("hidden") && !t.closest("aside") && !t.closest("#sh-queue")) toggle(false);
  });
  onSnapshot(showNow);
  api.GetSnapshot().then(showNow);
  return { toggle, refresh, isOpen: () => !panel.classList.contains("hidden") };
}
```

- [ ] **Step 4: Brancher dans le shell**

Dans `frontend/src/shell/shell.ts` :
- importer `mountQueuePanel` depuis `./queue` ;
- supprimer le bloc `<div id="sh-queue-pop">…</div>` du template, ainsi que les gestionnaires `sh-prev` et `sh-next`. Précédent et suivant restent accessibles par les flèches du clavier (`main.ts`) et le widget ;
- changer l'`aria-label` de `sh-queue` en « File d'attente » ;
- après `root.innerHTML = …`, ajouter `const queue = mountQueuePanel(root);` ;
- remplacer le gestionnaire `$("sh-queue").onclick` par :

```ts
  $("sh-queue").onclick = (e) => { e.stopPropagation(); toggle("sh-more", "sh-more-pop", false); queue.toggle(); $("sh-queue").setAttribute("aria-expanded", String(queue.isOpen())); };
```

- dans `closeAll`, retirer la ligne qui concernait `sh-queue-pop`.

Run: `cd frontend && npm test && npx tsc --noEmit` → Expected : PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src
git commit -m "feat(file): panneau « File d'attente » avec aléatoire et répétition

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 10: Partage façon story (1080×1920)

**Files:**
- Create: `frontend/src/share/story.ts`, `frontend/src/share/story.test.ts`
- Modify: `frontend/src/share/sharecard.ts`, `frontend/src/share/layout.ts`

**Interfaces:**
- Produces :
  - `STORY_W = 1080`, `STORY_H = 1920` (dans `layout.ts`)
  - `storyLayout(): { cover: { x: number; y: number; size: number; angle: number }; disc: { cx: number; cy: number; r: number } }`
  - `renderStory(input: ShareInput, s: Settings, tint: string): Promise<HTMLCanvasElement>`
- `openShareCard` garde sa signature et utilise désormais `renderStory`.

- [ ] **Step 1: Test de mise en page**

`frontend/src/share/story.test.ts` :

```ts
import { expect, test } from "vitest";
import { STORY_H, STORY_W } from "./layout";
import { storyLayout } from "./story";

test("format story 1080×1920", () => {
  expect([STORY_W, STORY_H]).toEqual([1080, 1920]);
});
test("pochette en haut, inclinée ; vinyle en bas à droite, rogné par le bord", () => {
  const { cover, disc } = storyLayout();
  expect(cover.y).toBeLessThan(STORY_H / 3);
  expect(cover.angle).toBeLessThan(0);
  expect(disc.cy).toBeGreaterThan(STORY_H * 0.6);
  expect(disc.cx + disc.r).toBeGreaterThan(STORY_W); // déborde à droite
  expect(disc.cy + disc.r).toBeGreaterThan(STORY_H * 0.9);
});
```

Run: `cd frontend && npm test` → Expected : FAIL.

- [ ] **Step 2: Implémenter**

Ajouter à `frontend/src/share/layout.ts` : `export const STORY_W = 1080;` et `export const STORY_H = 1920;`.

`frontend/src/share/story.ts` :

```ts
// Image story façon MD Vinyl : pochette inclinée en haut, vinyle en bas à droite, « VINYL » discret.
import type { Settings } from "../api";
import { labelColorFor, labelInk } from "../turntable/label";
import { fitText, STORY_H, STORY_W } from "./layout";
import type { ShareInput } from "./sharecard";

export function storyLayout() {
  return {
    cover: { x: 60, y: 140, size: 840, angle: -8 },
    disc: { cx: 760, cy: 1500, r: 520 },
  };
}

function loadImage(src: string): Promise<HTMLImageElement | null> {
  if (!src) return Promise.resolve(null);
  return new Promise((resolve) => {
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.onload = () => resolve(img);
    img.onerror = () => resolve(null);
    img.src = src;
  });
}

export async function renderStory(input: ShareInput, s: Settings, tint: string): Promise<HTMLCanvasElement> {
  const c = document.createElement("canvas");
  c.width = STORY_W;
  c.height = STORY_H;
  const g = c.getContext("2d")!;
  const { cover, disc } = storyLayout();
  const art = await loadImage(input.artworkUrl);

  // Fond teinté
  const bg = g.createLinearGradient(0, 0, 0, STORY_H);
  bg.addColorStop(0, tint);
  bg.addColorStop(1, "#1a1210");
  g.fillStyle = bg;
  g.fillRect(0, 0, STORY_W, STORY_H);

  // Pochette inclinée avec ombre
  g.save();
  g.translate(cover.x + cover.size / 2, cover.y + cover.size / 2);
  g.rotate((cover.angle * Math.PI) / 180);
  g.shadowColor = "rgba(0,0,0,.45)";
  g.shadowBlur = 60;
  g.shadowOffsetY = 30;
  if (art) g.drawImage(art, -cover.size / 2, -cover.size / 2, cover.size, cover.size);
  else { g.fillStyle = "#2a2220"; g.fillRect(-cover.size / 2, -cover.size / 2, cover.size, cover.size); }
  g.restore();

  // Vinyle
  g.save();
  g.shadowColor = "rgba(0,0,0,.5)";
  g.shadowBlur = 50;
  g.beginPath(); g.arc(disc.cx, disc.cy, disc.r, 0, Math.PI * 2);
  g.fillStyle = "#121010"; g.fill();
  g.restore();
  g.strokeStyle = "rgba(255,255,255,.05)";
  for (let rr = disc.r * 0.5; rr < disc.r; rr += 5) { g.beginPath(); g.arc(disc.cx, disc.cy, rr, 0, Math.PI * 2); g.stroke(); }

  // Étiquette
  const lr = disc.r * 0.46;
  const labelColor = labelColorFor(s.texture, s.texture === "picture" ? tint : null);
  const ink = labelInk(labelColor);
  g.beginPath(); g.arc(disc.cx, disc.cy, lr, 0, Math.PI * 2);
  g.fillStyle = labelColor; g.fill();
  g.strokeStyle = ink; g.lineWidth = 3;
  g.beginPath(); g.arc(disc.cx, disc.cy, lr * 0.92, 0, Math.PI * 2); g.stroke();
  g.save();
  g.translate(disc.cx, disc.cy);
  g.rotate(-0.5);
  g.fillStyle = ink;
  g.textAlign = "center";
  g.font = "800 46px -apple-system, sans-serif";
  g.fillText(fitText((t) => g.measureText(t).width, input.title, lr * 1.5), 0, -20);
  g.font = "500 30px -apple-system, sans-serif";
  g.fillText(fitText((t) => g.measureText(t).width, input.subtitle, lr * 1.4), 0, 40);
  g.restore();
  g.beginPath(); g.arc(disc.cx, disc.cy, 12, 0, Math.PI * 2);
  g.fillStyle = "#cfcac4"; g.fill();

  // Mention
  g.fillStyle = "rgba(255,255,255,.55)";
  g.font = "600 34px -apple-system, sans-serif";
  g.textAlign = "left";
  g.fillText("VINYL", 50, STORY_H - 60);
  return c;
}
```

Dans `frontend/src/share/sharecard.ts`, dans `openShareCard` :
- remplacer `const canvas = await renderCard(input, s);` par :

```ts
  const tint = (await imageTint(input.artworkUrl)) ?? s.accent;
  const canvas = await renderStory(input, s, tint);
```

- dans le `catch` du `toDataURL`, remplacer le second rendu par `(await renderStory({ ...input, artworkUrl: "" }, s, tint)).toDataURL("image/png")` ;
- la balise `<img>` de l'aperçu prend la classe `h-[70vh] rounded-[28px]` ;
- imports : `renderStory` depuis `./story`, `imageTint` depuis `../ui/theme`. `renderCard` reste exporté, inutilisé (supprimer s'il n'est plus importé nulle part : `grep -rn renderCard frontend/src`).

Run: `cd frontend && npm test && npx tsc --noEmit` → Expected : PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/src
git commit -m "feat(partage): image story 1080×1920 façon MD Vinyl

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

---

### Task 11: Vérification visuelle, build et recette

**Files:**
- Modify: `.superpowers/mock/service.ts` (hors dépôt, ignoré par git)

- [ ] **Step 1: Compléter le faux backend**

Dans `.superpowers/mock/service.ts`, ajouter à l'objet du `Proxy` :

```ts
  Stats: async () => ({ count: 34, since: "2026.10.14" }),
  Tracks: async () => ({ id: "a0", kind: "album", title: "In Utero (20th Anniversary Edition)", artist: "Nirvana", genre: "Alternative", year: "1993", artworkUrl: art, source: "applemusic",
    tracks: [["Serve the Servants", 217000], ["Scentless Apprentice", 228000], ["Heart-Shaped Box", 281000], ["Rape Me", 170000]].map(([t, d], i) => ({ id: "t" + i, number: i + 1, title: t, artist: "Nirvana", durationMs: d, loved: i === 2 })) }),
  Queue: async () => ({ shuffle: false, repeat: "all", note: "", upcoming: [{ id: "q1", number: 1, title: "Heart Flutters (feat. FlowerBoyDeMii)", artist: "Egofear", durationMs: 190000, loved: false }] }),
```

- [ ] **Step 2: Captures**

Construire le frontend simulé et prendre des captures en 1280×720. Utiliser la même config Vite simulée qu'en fin de plan précédent : `publicDir` et alias vers `.superpowers/mock`, avec la config écrite temporairement dans `frontend/.vite.mock.config.ts` puis supprimée. Routes :
- `#/platine` ;
- `#/platine`, en faisant défiler pour voir la bande stats (capture en 1280×1400) ;
- `#/bibliotheque/albums` ;
- `#/album/applemusic/a0`.

Comparer visuellement avec les fichiers suivants de `screen/` :
- `vinylvisual(withoutprice).png` ;
- `timevisual.png` ;
- `playlistvisual.jpg` ;
- `playlistalbum2.png`.

Corriger les écarts flagrants (positions, tailles) en restant dans les fichiers des tâches concernées. Pour la file d'attente, ouvrir le panneau via un paramètre de test, par exemple `?queue=1` lu dans `shell.ts`, ou bien la vérifier dans l'app réelle.

- [ ] **Step 3: Suite complète et build**

```bash
CGO_ENABLED=1 go test -race . ./internal/... && CGO_ENABLED=1 go vet . ./internal/...
cd frontend && npm test && npx tsc --noEmit && cd ..
CGO_ENABLED=1 wails3 package
```

Expected : tout passe ; `bin/vinyl.app` reconstruit.

- [ ] **Step 4: Recette manuelle (avec l'utilisateur)**

- [ ] Platine : étiquette noire, bras vertical au repos qui pivote à la lecture.
- [ ] Défiler sous la platine : compteur à rouleaux ; lancer un nouveau morceau → +1 avec animation.
- [ ] Étagère : molette et flèches font défiler le coverflow ; tranches lisibles ; clic central → page album.
- [ ] Page album : tri correct ; durées `3′37″` ; ♥ sur les titres aimés ; « Lecture aléatoire » active l'aléatoire.
- [ ] File d'attente : titres suivants ; aléatoire et répéter changent vraiment l'état dans Musique.
- [ ] Partage depuis la platine, une page album et le compteur : image verticale, Copier et Enregistrer fonctionnent.

- [ ] **Step 5: Commit final éventuel**

```bash
git add -A frontend/src
git commit -m "fix(interface): ajustements visuels après comparaison avec les captures MD Vinyl

Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>"
```

(Seulement si l'étape 2 a donné lieu à des corrections.)
