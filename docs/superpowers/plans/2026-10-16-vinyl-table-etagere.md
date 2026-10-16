# Platine : grande table et étagère de playlists — plan d'implémentation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Une grande table en bois occupe tout le bas de la platine, platine et pile 3D réduites posées dessus, et une petite étagère murale de playlists ouvre la vue Étagère au clic.

**Architecture:** Une fonction pure `layoutScene` calcule échelles et positions (platine, pile, étagère, ligne de la table) ; `views/platine.ts` les applique dans `fit()`. L'étagère est un composant isolé `ui/wallshelf.ts` (données injectées). Le service Go gagne `CurrentPlaylist(source)` (Apple Music : contexte de lecture ; Spotify : API web).

**Tech Stack:** TypeScript + Vitest (jsdom), Tailwind/CSS, Go 1.26 + Wails v3 (liaisons générées).

**Spec:** `docs/superpowers/specs/2026-10-16-vinyl-table-etagere-design.md`

## Global Constraints

- Branche : `feat/table-etagere`.
- Commits `type(portée): description` en français, sans aucune mention de Claude.
- Frontend : `cd frontend && npx tsc --noEmit && npx vitest run`.
- Go Mac : `CGO_ENABLED=1 go test -race . ./internal/...` ; Windows : `GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./internal/...`.
- Liaisons après ajout de méthode au service : `PATH="$PATH:$(go env GOPATH)/bin" wails3 generate bindings -clean=true -ts -i` (depuis la racine).
- Contrôle visuel : `bash .superpowers/mock/shoot.sh <dossier> "nom LxH #/platine"` (faux backend, Chrome sans affichage) ; supprimer `.superpowers/mock/dist` après.
- Étagère : `SHELF_MAX = 12` playlists au plus (11 tranches + 1 pochette en biais) ; infobulle « Ouvrir l'étagère » ; libellé « Ouvrir l'étagère des playlists ».
- Échelles : `TT_SCALE = 0.75` (platine), `PILE_SCALE = 0.7` (pile).
- Clic sur l'étagère : `localStorage["vinyl.shelf"] = "side"`, oubli de la position `source:playlists`, route `#/bibliotheque/playlists`.
- Textes de l'interface en français.

## Review Focus

1. Fenêtre très large et basse (ex. 1920×700) ou étroite et haute (1100×1000) : rien ne passe sous la colonne de commandes et la table touche toujours le bas — test Task 3 (tailles extrêmes).
2. Changement de source (Apple Music ↔ Spotify) pendant que la platine est affichée : l'étagère se recharge — test Task 4 (refresh appelé au changement de `source` dans les réglages).
3. `load()` qui répond après que la platine a été quittée (démonté) : aucune erreur, rien n'est écrit — test Task 2 (`destroy` avant la résolution).
4. Titre de playlist très long ou avec `<`/`&` : pas d'injection HTML, infobulle correcte — test Task 2 (titre `<b>&</b>`).
5. Double clic rapide sur l'étagère : une seule navigation — test Task 2 (`open` appelé une fois par clic, pas de second écouteur après `refresh`).

---

### Task 1: Playlist en cours (service Go)

**Files:**
- Modify: `internal/catalog/spotifyweb.go` (méthode `CurrentPlaylist`)
- Modify: `internal/app/service.go` (méthode `CurrentPlaylist`)
- Test: `internal/catalog/spotifyweb_test.go`, `internal/app/modes_test.go`

**Interfaces:**
- Produces: `func (c *SpotifyWeb) CurrentPlaylist(ctx context.Context) (string, error)` → URI `spotify:playlist:…` ou "" ; `func (s *VinylService) CurrentPlaylist(source string) (string, error)` → `"playlist:<PID>"` (Apple Music), URI (Spotify), "" sinon ; liaison TS `api.CurrentPlaylist(source: string): Promise<string>`.

- [ ] **Step 1: Write the failing tests**

`internal/catalog/spotifyweb_test.go` (ajout) :

```go
// Playlist en cours (GET /me/player) : seulement si le contexte est une playlist.
func TestSpotifyCurrentPlaylist(t *testing.T) {
	for body, want := range map[string]string{
		`{"context":{"type":"playlist","uri":"spotify:playlist:PL"}}`: "spotify:playlist:PL",
		`{"context":{"type":"album","uri":"spotify:album:AL"}}`:       "",
		`{"context":null}`: "",
	} {
		c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/me/player" {
				http.Error(w, r.URL.Path, 404)
				return
			}
			w.Write([]byte(body))
		})
		if got, err := c.CurrentPlaylist(context.Background()); err != nil || got != want {
			t.Errorf("%s → %q, %v", body, got, err)
		}
	}
}

// Rien ne joue (204, corps vide) : pas d'erreur, pas de playlist.
func TestSpotifyCurrentPlaylistNothingPlaying(t *testing.T) {
	c := newSpotifyTest(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })
	if got, err := c.CurrentPlaylist(context.Background()); err != nil || got != "" {
		t.Errorf("%q, %v", got, err)
	}
}
```

`internal/app/modes_test.go` (ajout) :

```go
// Playlist en cours, au format des éléments de la bibliothèque.
func TestCurrentPlaylistAppleMusic(t *testing.T) {
	p := &fakeResumer{saved: player.PlaybackContext{PlaylistID: "PL", TrackID: "4"}}
	s := serviceWith(p)
	if got, err := s.CurrentPlaylist("applemusic"); err != nil || got != "playlist:PL" {
		t.Errorf("%q, %v", got, err)
	}
	p.saved.PlaylistID = ""
	if got, _ := s.CurrentPlaylist("applemusic"); got != "" {
		t.Errorf("sans playlist : %q", got)
	}
	if got, _ := s.CurrentPlaylist("spotify"); got != "" { // pas de catalogue Spotify branché
		t.Errorf("spotify : %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/catalog/ -run CurrentPlaylist && CGO_ENABLED=1 go test ./internal/app/ -run CurrentPlaylist`
Expected: FAIL (`CurrentPlaylist` undefined).

- [ ] **Step 3: Implement**

`internal/catalog/spotifyweb.go` (après `Queue`) :

```go
// CurrentPlaylist : URI de la playlist en cours de lecture (GET /me/player), "" si rien ne joue ou si le
// contexte n'est pas une playlist.
func (c *SpotifyWeb) CurrentPlaylist(ctx context.Context) (string, error) {
	tok, err := c.Token(ctx)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, "GET", c.BaseURL+"/me/player", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode == 204 {
		return "", nil // rien ne joue
	}
	if resp.StatusCode != 200 {
		return "", spotifyError(resp)
	}
	var body struct {
		Context *struct {
			Type string `json:"type"`
			URI  string `json:"uri"`
		} `json:"context"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Context == nil || body.Context.Type != "playlist" {
		return "", nil
	}
	return body.Context.URI, nil
}
```

`internal/app/service.go` (après `PlayFromHistory`) :

```go
// CurrentPlaylist : playlist en cours de lecture, au format des éléments de la bibliothèque
// (« playlist:<id> » pour Apple Music, URI pour Spotify) ; "" si inconnue.
func (s *VinylService) CurrentPlaylist(source string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), callLimit)
	defer cancel()
	switch source {
	case "applemusic":
		cs, ok := s.watcher.Player("applemusic").(player.ContextSaver)
		if !ok {
			return "", nil
		}
		c, err := cs.SaveContext(ctx)
		if err != nil || c.PlaylistID == "" {
			return "", nil
		}
		return "playlist:" + c.PlaylistID, nil
	case "spotify":
		if w, ok := s.catalogs["spotify"].(*catalog.SpotifyWeb); ok {
			if id, err := w.CurrentPlaylist(ctx); err == nil {
				return id, nil
			}
		}
	}
	return "", nil
}
```

(Vérifier que le faux lecteur de `serviceWith` s'appelle bien « applemusic » ; sinon adapter le test au nom réel.)

Régénérer les liaisons (racine du dépôt).

- [ ] **Step 4: Run tests**

Run: `CGO_ENABLED=1 go test -race . ./internal/... && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet ./internal/... && cd frontend && npx tsc --noEmit`
Expected: tout passe ; `frontend/bindings/vinyl/internal/app/vinylservice.ts` contient `CurrentPlaylist`.

- [ ] **Step 5: Commit**

```bash
git add internal frontend/bindings
git commit -m "feat(platine): playlist en cours (Apple Music et Spotify)"
```

---

### Task 2: Étagère murale (`ui/wallshelf.ts`)

**Files:**
- Create: `frontend/src/ui/wallshelf.ts`, `frontend/src/ui/wallshelf.test.ts`
- Modify: `frontend/src/views/shelfcache.ts` (fonction `forget`), `frontend/src/styles.css` (styles `.vy-ws-*`)
- Test: `frontend/src/views/shelfcache.test.ts`

**Interfaces:**
- Consumes: `Item` (`frontend/src/api.ts` : `id, kind, title, subtitle, artworkUrl, url`), `spineColors(title, tint?)` (`views/shelf`), `imageTint(url)` (`ui/theme`), `go` (`router`), `recall/remember` (`views/shelfcache`).
- Produces: `SHELF_MAX = 12`, `SHELF_W = 330`, `SHELF_H = 150` ; `mountWallShelf(el: HTMLElement, deps: WallShelfDeps): { refresh(): void; destroy(): void }` avec `WallShelfDeps = { load(): Promise<Item[]>; current(): Promise<string>; open(): void }` ; `openPlaylistShelf(source: string, nav?: (route, param) => void): void` ; `forget(key: string): void` dans `views/shelfcache.ts`.

- [ ] **Step 1: Write the failing tests**

`frontend/src/views/shelfcache.test.ts` (ajout) :

```ts
test("forget efface la position mémorisée", () => {
  remember("spotify:playlists", { items: [], next: 0, position: 120 });
  forget("spotify:playlists");
  expect(recall("spotify:playlists")).toBeUndefined();
});
```
(ajouter `forget` à l'import existant.)

`frontend/src/ui/wallshelf.test.ts` :

```ts
import { expect, test, vi } from "vitest";
import type { Item } from "../api";
import { mountWallShelf, openPlaylistShelf, SHELF_MAX } from "./wallshelf";

const pl = (i: number, title = `Playlist ${i}`): Item => ({ id: `playlist:${i}`, kind: "playlist", title, subtitle: "", artworkUrl: `/art/${i}`, url: "" });
const flush = () => new Promise((r) => setTimeout(r, 0));
const deps = (items: Item[], current = "") => ({ load: vi.fn(() => Promise.resolve(items)), current: vi.fn(() => Promise.resolve(current)), open: vi.fn() });

test("12 playlists au plus : 11 tranches + 1 pochette en biais (la première par défaut)", async () => {
  const el = document.createElement("div");
  mountWallShelf(el, deps(Array.from({ length: 15 }, (_, i) => pl(i))));
  await flush();
  expect(SHELF_MAX).toBe(12);
  expect(el.querySelectorAll(".vy-ws-spine").length).toBe(11);
  expect(el.querySelector<HTMLElement>(".vy-ws-cover")!.style.backgroundImage).toContain("/art/0");
});

test("pochette en biais = playlist en cours si elle est sur l'étagère, jamais répétée en tranche", async () => {
  const el = document.createElement("div");
  mountWallShelf(el, deps(Array.from({ length: 5 }, (_, i) => pl(i)), "playlist:3"));
  await flush();
  expect(el.querySelector<HTMLElement>(".vy-ws-cover")!.style.backgroundImage).toContain("/art/3");
  const titles = [...el.querySelectorAll<HTMLElement>(".vy-ws-spine")].map((s) => s.title);
  expect(titles).not.toContain("Playlist 3");
  expect(titles.length).toBe(4);
});

test("playlist en cours absente de l'étagère : la première", async () => {
  const el = document.createElement("div");
  mountWallShelf(el, deps([pl(0), pl(1)], "playlist:99"));
  await flush();
  expect(el.querySelector<HTMLElement>(".vy-ws-cover")!.style.backgroundImage).toContain("/art/0");
});

test("clic n'importe où ou clavier : open, une fois par geste, même après refresh", async () => {
  const el = document.createElement("div");
  const d = deps([pl(0), pl(1)]);
  const shelf = mountWallShelf(el, d);
  await flush();
  shelf.refresh();
  await flush();
  el.querySelector<HTMLElement>(".vy-ws-spine")!.click();
  expect(d.open).toHaveBeenCalledTimes(1);
  el.querySelector<HTMLElement>(".vy-ws-board")!.click();
  el.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" }));
  el.dispatchEvent(new KeyboardEvent("keydown", { key: " " }));
  expect(d.open).toHaveBeenCalledTimes(4);
  expect(el.getAttribute("role")).toBe("button");
  expect(el.getAttribute("aria-label")).toBe("Ouvrir l'étagère des playlists");
  expect(el.title).toBe("Ouvrir l'étagère");
});

test("liste vide ou erreur : planche vide, toujours cliquable, sans message", async () => {
  for (const load of [() => Promise.resolve([] as Item[]), () => Promise.reject(new Error("403"))]) {
    const el = document.createElement("div");
    const open = vi.fn();
    mountWallShelf(el, { load, current: () => Promise.resolve(""), open });
    await flush();
    expect(el.querySelector(".vy-ws-board")).not.toBeNull();
    expect(el.querySelectorAll(".vy-ws-spine, .vy-ws-cover").length).toBe(0);
    expect(el.textContent?.trim()).toBe("");
    el.click();
    expect(open).toHaveBeenCalledTimes(1);
  }
});

test("titre piégé : pas d'injection HTML", async () => {
  const el = document.createElement("div");
  mountWallShelf(el, deps([pl(0), pl(1, "<b>&</b>")]));
  await flush();
  const spine = el.querySelector<HTMLElement>(".vy-ws-spine")!;
  expect(spine.querySelector("b")).toBeNull();
  expect(spine.title).toBe("<b>&</b>");
});

test("démontée avant la fin du chargement : rien n'est écrit", async () => {
  const el = document.createElement("div");
  let resolve!: (l: Item[]) => void;
  const shelf = mountWallShelf(el, { load: () => new Promise((r) => (resolve = r)), current: () => Promise.resolve(""), open: vi.fn() });
  shelf.destroy();
  resolve([pl(0), pl(1)]);
  await flush();
  expect(el.querySelectorAll(".vy-ws-spine").length).toBe(0);
});

test("openPlaylistShelf : vue étagère forcée, position oubliée, playlists", () => {
  localStorage.setItem("vinyl.shelf", "flat");
  const nav = vi.fn();
  openPlaylistShelf("spotify", nav);
  expect(localStorage.getItem("vinyl.shelf")).toBe("side");
  expect(nav).toHaveBeenCalledWith("bibliotheque", "playlists");
});
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd frontend && npx vitest run src/ui/wallshelf.test.ts src/views/shelfcache.test.ts`
Expected: FAIL (module `./wallshelf` et `forget` absents).

- [ ] **Step 3: Implement**

`frontend/src/views/shelfcache.ts` (ajout) :

```ts
export function forget(key: string): void {
  memory.delete(key);
}
```

`frontend/src/ui/wallshelf.ts` :

```ts
// Petite étagère murale au-dessus de la pile : les playlists de la source active, sur la tranche, et une
// pochette posée en biais (la playlist en cours, sinon la première). Toute l'étagère est un seul bouton
// vers la vue Étagère des playlists.
import type { Item } from "../api";
import { go, type Route } from "../router";
import { forget } from "../views/shelfcache";
import { spineColors } from "../views/shelf";
import { imageTint } from "./theme";

export const SHELF_MAX = 12;
export const SHELF_W = 330; // px de scène
export const SHELF_H = 150;

export interface WallShelfDeps { load(): Promise<Item[]>; current(): Promise<string>; open(): void; }

const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;

// Hauteur d'une tranche (px), légèrement variable mais toujours la même pour une playlist.
function spineHeight(id: string): number {
  let h = 2166136261;
  for (let i = 0; i < id.length; i++) h = Math.imul(h ^ id.charCodeAt(i), 16777619) >>> 0;
  return 96 + (h % 15); // 96 à 110
}

export function mountWallShelf(el: HTMLElement, deps: WallShelfDeps): { refresh(): void; destroy(): void } {
  let alive = true;
  let gen = 0;
  el.classList.add("vy-wallshelf");
  el.setAttribute("role", "button");
  el.tabIndex = 0;
  el.setAttribute("aria-label", "Ouvrir l'étagère des playlists");
  el.title = "Ouvrir l'étagère";
  el.onclick = () => deps.open();
  el.onkeydown = (e) => {
    if (e.key === "Enter" || e.key === " ") { e.preventDefault(); deps.open(); }
  };
  const books = document.createElement("div");
  books.className = "vy-ws-books";
  const board = document.createElement("div");
  board.className = "vy-ws-board";
  el.replaceChildren(books, board, Object.assign(document.createElement("div"), { className: "vy-ws-bracket vy-ws-bracket-l" }),
    Object.assign(document.createElement("div"), { className: "vy-ws-bracket vy-ws-bracket-r" }));

  const render = (list: Item[], currentId: string) => {
    const shown = list.slice(0, SHELF_MAX);
    const cover = shown.find((p) => p.id === currentId) ?? shown[0];
    const spines = shown.filter((p) => p !== cover);
    const nodes: HTMLElement[] = spines.map((p) => {
      const s = document.createElement("div");
      s.className = "vy-ws-spine";
      s.title = p.title;
      s.style.height = `${spineHeight(p.id)}px`;
      const paint = (tint?: string) => {
        const c = spineColors(p.title, tint);
        s.style.background = c.bg;
      };
      paint();
      if (p.artworkUrl) imageTint(p.artworkUrl).then((t) => { if (alive && t) paint(t); }).catch(() => {});
      return s;
    });
    if (cover) {
      const c = document.createElement("div");
      c.className = "vy-ws-cover";
      c.title = cover.title;
      if (cover.artworkUrl) c.style.backgroundImage = cssUrl(cover.artworkUrl);
      nodes.splice(Math.min(5, nodes.length), 0, c); // posée au milieu des tranches
    }
    books.replaceChildren(...nodes);
  };

  const refresh = () => {
    const mine = ++gen;
    Promise.all([deps.load().catch(() => [] as Item[]), deps.current().catch(() => "")])
      .then(([list, cur]) => { if (alive && mine === gen) render(list ?? [], cur ?? ""); });
  };
  refresh();
  return { refresh, destroy() { alive = false; } };
}

// Clic sur l'étagère : vue Étagère des playlists, au début, en mode étagère.
export function openPlaylistShelf(source: string, nav: (route: Route, param: string) => void = go): void {
  try { localStorage.setItem("vinyl.shelf", "side"); } catch { /* stockage indisponible */ }
  forget(`${source}:playlists`);
  nav("bibliotheque", "playlists");
}
```

`frontend/src/styles.css` (ajout, après les styles `.vy-recent-*`) :

```css
/* Étagère murale de playlists (platine) : tranches serrées, une pochette en biais, planche et équerres. */
.vy-wallshelf { position: absolute; width: 330px; height: 150px; transform-origin: 0 0; cursor: pointer; outline: none; transition: filter .2s; }
.vy-wallshelf:hover, .vy-wallshelf:focus-visible { filter: brightness(1.12); }
.vy-wallshelf:focus-visible .vy-ws-board { box-shadow: 0 0 0 3px var(--accent), 0 10px 16px rgba(0,0,0,.45); }
.vy-ws-books { position: absolute; left: 14px; right: 14px; bottom: 22px; height: 120px; display: flex; align-items: flex-end; gap: 2px; }
.vy-ws-spine { width: 15px; flex: none; border-radius: 2px 2px 1px 1px; box-shadow: inset -2px 0 0 rgba(0,0,0,.18), inset 1px 0 0 rgba(255,255,255,.12), 1px 0 2px rgba(0,0,0,.35); }
.vy-ws-cover { width: 92px; height: 92px; flex: none; margin: 0 10px 0 6px; border-radius: 3px; background: #2a2220 center / cover;
  transform: rotate(-9deg); transform-origin: 0 100%; box-shadow: 0 6px 12px rgba(0,0,0,.45), inset 0 0 0 1px rgba(255,255,255,.12); }
.vy-ws-board { position: absolute; left: 0; right: 0; bottom: 12px; height: 11px; border-radius: 3px;
  background: linear-gradient(180deg, #a0714f, #7a5238 55%, #5d3b26); box-shadow: 0 10px 16px rgba(0,0,0,.45); }
.vy-ws-bracket { position: absolute; bottom: 0; width: 7px; height: 14px; background: #3b2618; border-radius: 0 0 2px 2px; }
.vy-ws-bracket-l { left: 40px; }
.vy-ws-bracket-r { right: 40px; }
```

- [ ] **Step 4: Run tests**

Run: `cd frontend && npx tsc --noEmit && npx vitest run`
Expected: tout passe.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/ui/wallshelf.ts frontend/src/ui/wallshelf.test.ts frontend/src/views/shelfcache.ts frontend/src/views/shelfcache.test.ts frontend/src/styles.css
git commit -m "feat(platine): étagère murale de playlists"
```

---

### Task 3: Disposition de la scène (`views/scene.ts`)

**Files:**
- Create: `frontend/src/views/scene.ts`, `frontend/src/views/scene.test.ts`

**Interfaces:**
- Consumes: `STAGE_W = 960`, `STAGE_H = 660` (`turntable/turntable`), `SHELF_W`, `SHELF_H` (Task 2), pile `500 × 460` (zone de `ui/recentpile`).
- Produces: `TT_SCALE = 0.75`, `PILE_SCALE = 0.7`, `ROOM = { w: 1180, h: 760 }`, `TABLE_Y = 560` ; `layoutScene(w: number, h: number): SceneLayout` avec `SceneLayout = { k: number; stage: Placed; pile: Placed; shelf: Placed; tableTop: number }`, `Placed = { x: number; y: number; scale: number }` (px dans `#pl-wrap`).

- [ ] **Step 1: Write the failing test** — `frontend/src/views/scene.test.ts`

```ts
import { expect, test } from "vitest";
import { STAGE_H, STAGE_W } from "../turntable/turntable";
import { SHELF_H, SHELF_W } from "../ui/wallshelf";
import { layoutScene, PILE_SCALE, TT_SCALE } from "./scene";

const PILE = { w: 500, h: 460 };
// Tailles de #pl-wrap (fenêtre moins la colonne de commandes de 140 px), y compris les extrêmes.
const SIZES: [number, number][] = [[1300, 900], [1780, 1080], [960, 700], [1780, 700], [960, 1000], [600, 400]];

test.each(SIZES)("%i×%i : tout tient dans la zone, posé sur la table", (w, h) => {
  const L = layoutScene(w, h);
  const right = (p: { x: number; scale: number }, width: number) => p.x + width * p.scale;
  // Rien ne déborde à droite (sous la colonne de commandes) ni à gauche.
  for (const [p, width] of [[L.stage, STAGE_W], [L.pile, PILE.w], [L.shelf, SHELF_W]] as const) {
    expect(p.x).toBeGreaterThanOrEqual(0);
    expect(right(p, width)).toBeLessThanOrEqual(w + 0.5);
  }
  expect(L.stage.scale).toBeCloseTo(L.k * TT_SCALE, 6);
  expect(L.pile.scale).toBeCloseTo(L.k * PILE_SCALE, 6);
  // Platine et pile reposent sur la table : leur bas passe sous la ligne de la table.
  expect(L.stage.y + STAGE_H * L.stage.scale).toBeGreaterThan(L.tableTop);
  expect(L.pile.y + PILE.h * L.pile.scale).toBeGreaterThan(L.tableTop);
  expect(L.tableTop).toBeLessThan(h);
  // L'étagère est au mur : au-dessus de la table et de la pile, à droite de la platine.
  expect(L.shelf.y + SHELF_H * L.shelf.scale).toBeLessThan(L.tableTop);
  expect(L.shelf.x).toBeGreaterThan(L.stage.x + STAGE_W * L.stage.scale * 0.6);
});

test("la scène est centrée et grandit avec la fenêtre", () => {
  const small = layoutScene(960, 700), big = layoutScene(1780, 1080);
  expect(big.k).toBeGreaterThan(small.k);
  const L = layoutScene(1780, 700); // très large : marges égales à gauche et à droite de la scène
  const left = L.stage.x, rightEdge = L.shelf.x + SHELF_W * L.shelf.scale;
  expect(Math.abs(left - (1780 - rightEdge))).toBeLessThan(200);
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/views/scene.test.ts`
Expected: FAIL (module `./scene` absent).

- [ ] **Step 3: Implement** — `frontend/src/views/scene.ts`

```ts
// Disposition de la platine : une « pièce » de taille fixe (ROOM, px de scène) mise à l'échelle dans la
// zone disponible, centrée. La table occupe le bas (à partir de TABLE_Y) ; platine et pile, réduites, y
// reposent ; l'étagère est au mur, au-dessus de la pile.
import { STAGE_H } from "../turntable/turntable";
import { SHELF_W } from "../ui/wallshelf";

export const TT_SCALE = 0.75;
export const PILE_SCALE = 0.7;
export const ROOM = { w: 1180, h: 760 };
export const TABLE_Y = 560;
const PILE_H = 460;

export interface Placed { x: number; y: number; scale: number; }
export interface SceneLayout { k: number; stage: Placed; pile: Placed; shelf: Placed; tableTop: number; }

// Positions dans la pièce (px de scène).
const STAGE_AT = { x: 10, y: TABLE_Y + 70 - STAGE_H * TT_SCALE }; // le bas de la platine sur la table
const PILE_AT = { x: 780, y: TABLE_Y + 40 - PILE_H * PILE_SCALE }; // la base de la pile sur la table
const SHELF_AT = { x: ROOM.w - SHELF_W - 30, y: 90 };

export function layoutScene(w: number, h: number): SceneLayout {
  const k = Math.min(w / ROOM.w, h / ROOM.h);
  const ox = (w - ROOM.w * k) / 2, oy = (h - ROOM.h * k) / 2;
  const at = (p: { x: number; y: number }, scale: number): Placed => ({ x: ox + p.x * k, y: oy + p.y * k, scale });
  return {
    k,
    stage: at(STAGE_AT, k * TT_SCALE),
    pile: at(PILE_AT, k * PILE_SCALE),
    shelf: at(SHELF_AT, k),
    tableTop: oy + TABLE_Y * k,
  };
}
```

- [ ] **Step 4: Run tests**

Run: `cd frontend && npx tsc --noEmit && npx vitest run src/views/scene.test.ts`
Expected: PASS (6 tailles + centrage). Si une taille échoue, ajuster les constantes de position (pas les tests) en gardant les invariants.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/views/scene.ts frontend/src/views/scene.test.ts
git commit -m "feat(platine): disposition de la scène (table, platine, pile, étagère)"
```

---

### Task 4: Platine — table pleine largeur, scène réduite, étagère branchée

**Files:**
- Modify: `frontend/src/views/platine.ts` (gabarit, `fit`, étagère), `frontend/src/ui/recentpile.ts` (retrait de la petite table, base de la pile rétablie), `frontend/src/ui/recentpile.test.ts` (test de la petite table retiré), `frontend/src/styles.css` (styles `.vy-table*` retirés, `.vy-room-table` ajouté)
- Test: `frontend/src/views/platine-shelf.test.ts`

**Interfaces:**
- Consumes: `layoutScene` (Task 3), `mountWallShelf`, `openPlaylistShelf` (Task 2), `api.CurrentPlaylist` (Task 1), `activeSource()` (`api.ts`), `onSettings` (`api.ts`).
- Produces: `wireShelf(el, deps: { source(): Promise<string>; library(source: string): Promise<Item[]>; current(source: string): Promise<string>; nav?: (route, param) => void; onSettings(cb: (s: { source: string }) => void): () => void })` → `{ destroy(): void }` exporté par `views/platine.ts` (testable sans Wails).

- [ ] **Step 1: Write the failing test** — `frontend/src/views/platine-shelf.test.ts`

```ts
import { expect, test, vi } from "vitest";
import type { Item } from "../api";
import { wireShelf } from "./platine";

const pl = (i: number): Item => ({ id: `playlist:${i}`, kind: "playlist", title: `P${i}`, subtitle: "", artworkUrl: "", url: "" });
const flush = () => new Promise((r) => setTimeout(r, 0));

test("l'étagère charge les playlists de la source active et se recharge au changement de source", async () => {
  let source = "applemusic";
  let notify!: (s: { source: string }) => void;
  const library = vi.fn((s: string) => Promise.resolve(s === "applemusic" ? [pl(1), pl(2)] : [pl(7), pl(8), pl(9)]));
  const el = document.createElement("div");
  wireShelf(el, {
    source: () => Promise.resolve(source), library, current: () => Promise.resolve(""),
    onSettings: (cb) => { notify = cb; return () => {}; },
  });
  await flush();
  expect(library).toHaveBeenLastCalledWith("applemusic");
  expect(el.querySelectorAll(".vy-ws-spine").length).toBe(1);
  source = "spotify";
  notify({ source: "spotify" });
  await flush();
  expect(library).toHaveBeenLastCalledWith("spotify");
  expect(el.querySelectorAll(".vy-ws-spine").length).toBe(2);
});

test("clic : vue Étagère des playlists de la source active", async () => {
  const nav = vi.fn();
  const el = document.createElement("div");
  wireShelf(el, { source: () => Promise.resolve("spotify"), library: () => Promise.resolve([]), current: () => Promise.resolve(""), nav, onSettings: () => () => {} });
  await flush();
  el.click();
  await flush();
  expect(nav).toHaveBeenCalledWith("bibliotheque", "playlists");
});
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/views/platine-shelf.test.ts`
Expected: FAIL (`wireShelf` non exporté).

- [ ] **Step 3: Implement**

`frontend/src/views/platine.ts` :

1. Imports : `import { layoutScene } from "./scene";`, `import { mountWallShelf, openPlaylistShelf } from "../ui/wallshelf";`, et `activeSource`, `type Item` depuis `../api` (ajoutés à l'import existant), `type Route` depuis `../router`.
2. Supprimer les constantes `SCENE_W` et `RECENT` (remplacées par `layoutScene`).
3. Gabarit : dans `<section class="relative h-screen">`, juste avant `<div id="pl-wrap" …>`, ajouter `<div id="pl-table" class="vy-room-table" aria-hidden="true"></div>` ; dans `#pl-wrap`, après `#pl-recent`, ajouter `<div id="pl-shelf"></div>`.
4. Remplacer le corps de `fit` par :

```ts
  const fit = () => {
    const wrap = $("pl-wrap").getBoundingClientRect();
    const L = layoutScene(wrap.width, wrap.height);
    const s = $("pl-scaled");
    s.style.transform = `scale(${L.stage.scale})`;
    s.style.left = `${L.stage.x}px`;
    s.style.top = `${L.stage.y}px`;
    const r = $("pl-recent");
    r.style.left = `${L.pile.x}px`;
    r.style.top = `${L.pile.y}px`;
    r.style.width = `${500 * L.pile.scale}px`;
    r.style.height = `${460 * L.pile.scale}px`;
    recent?.setScale(L.pile.scale);
    const sh = $("pl-shelf");
    sh.style.left = `${L.shelf.x}px`;
    sh.style.top = `${L.shelf.y}px`;
    sh.style.transform = `scale(${L.shelf.scale})`;
    $("pl-table").style.top = `${L.tableTop}px`;
  };
```

5. Après le montage de la pile (`recent = mountRecentPile(...)`) :

```ts
  const shelf = wireShelf($("pl-shelf"), {
    source: activeSource,
    library: (src) => api.Library(src, "playlists", 0).then((p) => (p.items ?? []) as Item[]),
    current: (src) => api.CurrentPlaylist(src),
    onSettings,
  });
```
et dans la fonction de nettoyage retournée : `shelf.destroy();`.

6. Exporter, en bas du fichier :

```ts
// Étagère murale branchée sur la source active : rechargée quand la source change dans les réglages ;
// un clic ouvre la vue Étagère des playlists de cette source.
export function wireShelf(el: HTMLElement, deps: {
  source(): Promise<string>;
  library(source: string): Promise<Item[]>;
  current(source: string): Promise<string>;
  nav?: (route: Route, param: string) => void;
  onSettings(cb: (s: { source: string }) => void): () => void;
}): { destroy(): void } {
  let lastSource = "";
  let prev: string | null = null; // source au dernier chargement (réglage « source » au dernier changement)
  const shelf = mountWallShelf(el, {
    load: () => deps.source().then((s) => { lastSource = s; prev ??= s; return deps.library(s); }),
    current: () => deps.source().then((s) => deps.current(s)),
    open: () => void deps.source().then((s) => openPlaylistShelf(s, deps.nav)).catch(() => openPlaylistShelf(lastSource, deps.nav)),
  });
  // Réglages modifiés : rechargement seulement si la source a changé (pas pour le craquement, etc.).
  const off = deps.onSettings((s) => {
    if (prev !== null && s.source !== prev) shelf.refresh();
    prev = s.source;
  });
  return { destroy() { off(); shelf.destroy(); } };
}
```

`frontend/src/ui/recentpile.ts` : retirer la petite table (constantes `PILE_BASE`, `TOP_T`, `LEG*`, `OVER`, `TABLE_TOP`, `PIVOT`, `legAt`, `IN`, `LEGS`, fonctions `face` et `table`, l'appel `pile.appendChild(table())`, l'import `boxMatrices`/`Box`) et rétablir :

```ts
// Vue de la pile : un peu d'en haut (point de fuite au-dessus, à 40 % de la largeur).
const VIEW: PileView = { width: PILE_W, height: 460, depth: PILE_W, slab: SLAB, eye: { x: PILE_W * 0.4, y: -70 }, perspective: 1800 };
```

`frontend/src/ui/recentpile.test.ts` : supprimer le test « une table porte la pile ».

`frontend/src/styles.css` : supprimer le bloc `/* Table sous la pile … */` (`.vy-table*`) et ajouter :

```css
/* Grande table de la platine : toute la largeur, du dessus (ligne calculée par layoutScene) au bas de page. */
.vy-room-table { position: absolute; left: 0; right: 0; bottom: 0; pointer-events: none;
  background: linear-gradient(180deg, #a0714f 0, #8d6243 10px, #7a5238 26px, #6b4630 60%, #5a3a27 100%),
    repeating-linear-gradient(90deg, rgba(255,255,255,.03) 0 3px, transparent 3px 14px, rgba(0,0,0,.035) 14px 16px, transparent 16px 37px);
  background-blend-mode: normal, overlay;
  box-shadow: inset 0 2px 0 rgba(255,230,200,.25), inset 0 14px 22px -14px rgba(0,0,0,.35); }
```

(`boxMatrices` et son test dans `pile3d` restent : utiles et testés.)

- [ ] **Step 4: Run tests and visual check**

Run: `cd frontend && npx tsc --noEmit && npx vitest run`
Expected: tout passe.

Run (racine) : `bash .superpowers/mock/shoot.sh /private/tmp/claude-501/-Users-pierre-Document-on-Mac-projet-git-vinyl/0ee312e2-09b9-484f-b161-385fe819a9a2/scratchpad/shots "p1440 1440x900 #/platine" "p1920 1920x1080 #/platine" "p1100 1100x700 #/platine"` puis lire les trois images.
Expected : table sur toute la largeur en bas ; platine et pile posées dessus, plus petites ; étagère au mur au-dessus de la pile, à gauche de la colonne de commandes ; rien ne se chevauche. Ajuster au besoin les constantes de `views/scene.ts` (positions) et relancer `vitest`. Puis `rm -rf .superpowers/mock/dist`.

- [ ] **Step 5: Commit**

```bash
git add frontend/src
git commit -m "feat(platine): grande table pleine largeur, platine et pile réduites, étagère de playlists"
```

---

### Task 5: Vérification finale et fabrication

**Files:** aucun (sauf corrections trouvées).

- [ ] **Step 1: Suites complètes**

Run: `CGO_ENABLED=1 go test -race . ./internal/... && GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go vet . ./internal/... && cd frontend && npx tsc --noEmit && npx vitest run`
Expected: tout passe.

- [ ] **Step 2: Fabrication et démarrage**

Run: `CGO_ENABLED=1 PATH="$PATH:$(go env GOPATH)/bin" wails3 package` puis lancer `./bin/vinyl.app/Contents/MacOS/vinyl` 6 s (en gardant son PID pour l'arrêter, jamais `pkill`).
Expected: fabrication OK, l'app reste lancée, pas d'erreur dans la sortie.

- [ ] **Step 3: Commit (si corrections)**

```bash
git add -A frontend internal
git commit -m "fix(platine): corrections de la vérification finale"
```
