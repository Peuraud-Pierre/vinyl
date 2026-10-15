// Pile des derniers morceaux écoutés, à droite du disque : pochettes couchées les unes sur les autres
// après avoir lu le disque, vues de face. On voit leurs tranches (couleur de la pochette, titre),
// du bas vers le haut ; la plus récente est en haut et sa pochette est visible, posée à plat.
import type { HistoryEntry } from "../api";
import { escapeHtml } from "../views/grid";
import { spineColors } from "../views/shelf";
import { imageTint } from "./theme";

export const RECENT_MAX = 7;
export const PILE_W = 290; // largeur d'une pochette (px de scène)
export const SLAB = 24; // épaisseur d'une tranche

// Décalage et légère inclinaison propres à chaque morceau (toujours les mêmes) : une pile naturelle
// qui ne bouge pas quand un nouveau disque est posé dessus.
function jitter(id: string): { dx: number; rot: number } {
  let h = 2166136261;
  for (let i = 0; i < id.length; i++) h = Math.imul(h ^ id.charCodeAt(i), 16777619) >>> 0;
  return { dx: (h % 15) - 7, rot: (((h >>> 6) % 25) - 12) / 10 }; // ±7 px, ±1,2°
}

const key = (e: HistoryEntry) => `${e.source}:${e.trackId}`;
const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;
const tints = new Map<string, string | null>(); // teinte de chaque pochette, calculée une fois

export interface RecentPile { update(list: HistoryEntry[]): void; }

export function mountRecentPile(root: HTMLElement, deps: { play(e: HistoryEntry): void }): RecentPile {
  let top = "";
  const paint = (slab: HTMLElement, it: HistoryEntry) => {
    const c = spineColors(it.title, tints.get(key(it)) ?? undefined);
    slab.style.background = c.bg;
    slab.style.color = c.ink;
  };
  return {
    update(list) {
      const shown = list.slice(0, RECENT_MAX);
      const n = shown.length;
      const newTop = shown[0] ? key(shown[0]) : "";
      const dropped = top !== "" && newTop !== "" && newTop !== top; // un disque vient d'être posé
      top = newTop;
      root.innerHTML = "";
      // Du plus ancien (en bas) au plus récent (en haut).
      for (let level = 0; level < n; level++) {
        const it = shown[n - 1 - level]!;
        const { dx, rot } = jitter(key(it));
        const b = document.createElement("button");
        b.className = "vy-recent";
        b.dataset.track = it.trackId;
        b.setAttribute("aria-label", `${it.title} — ${it.artist}`);
        b.title = `${it.title} — ${it.artist}`;
        b.style.bottom = `${level * SLAB}px`;
        b.style.transform = `translateX(${dx}px) rotate(${rot}deg)`;
        b.innerHTML = `<b>${escapeHtml(it.title)}</b><span>${escapeHtml(it.artist)}</span>`;
        b.onclick = () => deps.play(it);
        paint(b, it);
        root.appendChild(b);
        if (!tints.has(key(it)) && it.artworkUrl) {
          tints.set(key(it), null);
          imageTint(it.artworkUrl).then((t) => { tints.set(key(it), t); if (b.isConnected) paint(b, it); });
        }
      }
      if (n === 0) return;
      // Pochette du dessus, posée à plat : vue en perspective au-dessus de la pile.
      const last = shown[0]!;
      const lid = document.createElement("button");
      lid.className = "vy-recent-lid";
      lid.setAttribute("aria-label", `${last.title} — ${last.artist}`);
      lid.title = `${last.title} — ${last.artist}`;
      lid.style.bottom = `${n * SLAB}px`;
      const j = jitter(newTop);
      lid.style.left = `${j.dx}px`;
      lid.style.transform = `rotate(${j.rot}deg) perspective(900px) rotateX(74deg)`; // même inclinaison que sa tranche
      if (last.artworkUrl) lid.style.backgroundImage = cssUrl(last.artworkUrl);
      lid.onclick = () => deps.play(last);
      root.appendChild(lid);
      if (dropped) {
        // Posé depuis le disque : il arrive d'en haut à gauche et se dépose sur la pile.
        const from = { opacity: 0, translate: "-140px -70px" };
        const to = { opacity: 1, translate: "0 0" };
        [root.lastElementChild!.previousElementSibling, lid].forEach((el) =>
          (el as HTMLElement | null)?.animate?.([from, to], { duration: 650, easing: "cubic-bezier(.2,.75,.2,1)" }));
      }
    },
  };
}
