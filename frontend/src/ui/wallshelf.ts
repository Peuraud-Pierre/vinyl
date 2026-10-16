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

const div = (className: string) => Object.assign(document.createElement("div"), { className });

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
  const books = div("vy-ws-books");
  el.replaceChildren(books, div("vy-ws-board"), div("vy-ws-bracket vy-ws-bracket-l"), div("vy-ws-bracket vy-ws-bracket-r"));

  const render = (list: Item[], currentId: string) => {
    const shown = list.slice(0, SHELF_MAX);
    const cover = shown.find((p) => p.id === currentId) ?? shown[0];
    const nodes: HTMLElement[] = shown.filter((p) => p !== cover).map((p) => {
      const s = div("vy-ws-spine");
      s.title = p.title;
      s.style.height = `${spineHeight(p.id)}px`;
      const paint = (tint?: string) => { s.style.background = spineColors(p.title, tint).bg; };
      paint();
      if (p.artworkUrl) imageTint(p.artworkUrl).then((t) => { if (alive && t) paint(t); }).catch(() => {});
      return s;
    });
    if (cover) {
      const c = div("vy-ws-cover");
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
