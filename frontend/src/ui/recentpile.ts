// Pile des derniers morceaux écoutés, à droite du disque : comme des pochettes posées les unes sur
// les autres après avoir lu le disque. La plus récente est au-dessus ; un clic la relance.
import type { HistoryEntry } from "../api";
import { escapeHtml, initials } from "../views/grid";

export const RECENT_MAX = 7;
const SLEEVE = 150; // côté d'une pochette (px de scène)

// Décalage et angle propres à chaque morceau (toujours les mêmes) : une pile naturelle qui ne
// tressaute pas quand un nouveau disque est posé dessus.
function jitter(id: string): { dx: number; dy: number; rot: number } {
  let h = 2166136261;
  for (let i = 0; i < id.length; i++) h = Math.imul(h ^ id.charCodeAt(i), 16777619) >>> 0;
  return { dx: (h % 25) - 12, dy: ((h >>> 5) % 17) - 8, rot: ((h >>> 9) % 19) - 9 };
}

const key = (e: HistoryEntry) => `${e.source}:${e.trackId}`;
const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;

export interface RecentPile { update(list: HistoryEntry[]): void; }

export function mountRecentPile(root: HTMLElement, deps: { play(e: HistoryEntry): void }): RecentPile {
  let top = "";
  return {
    update(list) {
      const shown = list.slice(0, RECENT_MAX);
      const newTop = shown[0] ? key(shown[0]) : "";
      const dropped = top !== "" && newTop !== "" && newTop !== top; // un disque vient d'être posé
      top = newTop;
      root.innerHTML = "";
      // Du dessous vers le dessus : la plus récente en dernier, donc au-dessus.
      shown.slice().reverse().forEach((it, j) => {
        const level = shown.length - 1 - j; // 0 = dessus
        const { dx, dy, rot } = jitter(key(it));
        const b = document.createElement("button");
        b.className = "vy-recent";
        b.dataset.track = it.trackId;
        b.setAttribute("aria-label", `${it.title} — ${it.artist}`);
        b.title = `${it.title} — ${it.artist}`;
        b.style.transform = `translate(${dx}px, ${dy}px) rotate(${rot}deg)`;
        b.innerHTML = `<div class="vy-recent-cover" style="width:${SLEEVE}px;height:${SLEEVE}px;${it.artworkUrl ? `background-image:${escapeHtml(cssUrl(it.artworkUrl))}` : ""}">${it.artworkUrl ? "" : `<span>${escapeHtml(initials(it.title))}</span>`}</div>`;
        b.onclick = () => deps.play(it);
        root.appendChild(b);
        if (dropped && level === 0) {
          // Posé depuis le disque, à gauche : il glisse et se dépose sur la pile.
          b.firstElementChild?.animate?.([
            { opacity: 0, transform: "translate(-170px, -50px) rotate(-24deg) scale(1.05)" },
            { opacity: 1, transform: "none" },
          ], { duration: 650, easing: "cubic-bezier(.2,.75,.2,1)" });
        }
      });
    },
  };
}
