// Pile des derniers morceaux écoutés, à droite du disque : pochettes couchées les unes sur les autres
// après avoir lu le disque, vues de face. On voit leurs tranches (couleur de la pochette, titre),
// du bas vers le haut ; la plus récente est en haut et sa pochette est visible, posée à plat.
import type { HistoryEntry } from "../api";
import { escapeHtml } from "../views/grid";
import { spineColors } from "../views/shelf";
import { imageTint } from "./theme";

export const RECENT_MAX = 7;
export const PILE_W = 500; // largeur d'une pochette (px de scène)
export const SLAB = 20; // épaisseur d'une tranche
export const PILE_TURN = -20; // la pile a pivoté sur la table : côté gauche plus loin (demande utilisateur)

// Décalage et rotation sur la table propres à chaque morceau (« au hasard », mais toujours les mêmes) :
// une pile posée en vrac qui ne bouge pas quand un nouveau disque est posé dessus.
function jitter(id: string): { dx: number; yaw: number } {
  let h = 2166136261;
  for (let i = 0; i < id.length; i++) h = Math.imul(h ^ id.charCodeAt(i), 16777619) >>> 0;
  return { dx: (h % 15) - 7, yaw: ((h >>> 6) % 19) - 9 }; // ±7 px, ±9°
}

const key = (e: HistoryEntry) => `${e.source}:${e.trackId}`;
const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;
const tints = new Map<string, string | null>(); // teinte de chaque pochette, calculée une fois

export interface RecentPile { update(list: HistoryEntry[]): void; setScale(k: number): void; }

const PERSPECTIVE = 1800; // px de scène ; vue un peu d'en haut (perspective-origin dans styles.css)

export function mountRecentPile(root: HTMLElement, deps: { play(e: HistoryEntry): void }): RecentPile {
  let top = "";
  let scale = 1;
  let zoom: HTMLElement | null = null;
  // Même montage que l'étagère, dont la 3D s'affiche bien dans WebKit : la perspective est sur un
  // conteneur non transformé (root), l'échelle sur le niveau en preserve-3d juste en dessous.
  const applyScale = () => {
    root.style.perspective = `${PERSPECTIVE * scale}px`;
    if (zoom) zoom.style.scale = String(scale);
  };
  const paint = (slab: HTMLElement, it: HistoryEntry) => {
    const c = spineColors(it.title, tints.get(key(it)) ?? undefined);
    slab.style.background = c.bg;
    slab.style.color = c.ink;
  };
  return {
    setScale(k) { scale = k; applyScale(); },
    update(list) {
      const shown = list.slice(0, RECENT_MAX);
      const n = shown.length;
      const newTop = shown[0] ? key(shown[0]) : "";
      const dropped = top !== "" && newTop !== "" && newTop !== top; // un disque vient d'être posé
      top = newTop;
      root.innerHTML = "";
      root.classList.add("vy-recent-scene");
      const pile = document.createElement("div");
      pile.className = "vy-recent-3d";
      root.appendChild(pile);
      zoom = pile;
      applyScale();
      let topDisc: HTMLElement | null = null;
      // Du plus ancien (en bas) au plus récent (en haut). Chaque disque est un petit volume :
      // tranche de face, côté droit (visible puisque la pile a pivoté) et pochette posée à plat dessus
      // (visible là où le disque dépasse de ceux qui le recouvrent).
      for (let level = 0; level < n; level++) {
        const it = shown[n - 1 - level]!;
        const { dx, yaw } = jitter(key(it));
        const d = document.createElement("div");
        d.className = "vy-recent-disc";
        d.style.bottom = `${level * SLAB}px`;
        // La pile a pivoté sur la table (PILE_TURN) ; chaque disque y ajoute son propre écart.
        d.style.transform = `translateX(${dx}px) rotateY(${PILE_TURN + yaw}deg)`;
        const b = document.createElement("button");
        b.className = "vy-recent";
        b.dataset.track = it.trackId;
        b.setAttribute("aria-label", `${it.title} — ${it.artist}`);
        b.title = `${it.title} — ${it.artist}`;
        b.innerHTML = `<b>${escapeHtml(it.title)}</b><span>${escapeHtml(it.artist)}</span>`;
        b.onclick = () => deps.play(it);
        const side = document.createElement("div");
        side.className = "vy-recent-side";
        // Côté plus sombre par la couleur : un filtre aplatirait la 3D dans WebKit.
        const paintAll = () => {
          paint(b, it);
          side.style.background = `color-mix(in srgb, ${spineColors(it.title, tints.get(key(it)) ?? undefined).bg} 62%, #000)`;
        };
        paintAll();
        const lid = document.createElement("button");
        lid.className = "vy-recent-lid";
        lid.tabIndex = -1; // la tranche suffit au clavier
        lid.setAttribute("aria-label", `${it.title} — ${it.artist}`);
        lid.title = `${it.title} — ${it.artist}`;
        if (it.artworkUrl) lid.style.backgroundImage = cssUrl(it.artworkUrl);
        lid.onclick = () => deps.play(it);
        d.append(b, side, lid);
        pile.appendChild(d);
        topDisc = d;
        if (!tints.has(key(it)) && it.artworkUrl) {
          tints.set(key(it), null);
          imageTint(it.artworkUrl).then((t) => { tints.set(key(it), t); if (b.isConnected) paintAll(); });
        }
      }
      if (n === 0) return;
      if (dropped) {
        // Posé depuis le disque : il arrive d'en haut à gauche et se dépose sur la pile.
        const from = { opacity: 0, translate: "-140px -70px" };
        const to = { opacity: 1, translate: "0 0" };
        topDisc?.animate?.([from, to], { duration: 650, easing: "cubic-bezier(.2,.75,.2,1)" });
      }
    },
  };
}
