// Pile des derniers morceaux écoutés, à droite du disque : pochettes couchées les unes sur les autres
// après avoir lu le disque, vues de face. On voit leurs tranches (couleur de la pochette, titre),
// du bas vers le haut ; la plus récente est en haut et sa pochette est visible, posée à plat.
import type { HistoryEntry } from "../api";
import { escapeHtml } from "../views/grid";
import { spineColors } from "../views/shelf";
import { boxMatrices, faceMatrices, toCss, type Box, type PileView } from "./pile3d";
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

// Vue de la pile : un peu d'en haut (point de fuite au-dessus, à 40 % de la largeur). La pile repose sur
// une table (plateau à PILE_BASE, pieds jusqu'en bas de la zone).
const PILE_BASE = 345;
const VIEW: PileView = { width: PILE_W, height: PILE_BASE, depth: PILE_W, slab: SLAB, eye: { x: PILE_W * 0.4, y: PILE_BASE - 530 }, perspective: 1800 };

// Table : plateau qui déborde des disques, quatre pieds ; tournée comme la pile.
const TOP_T = 24, LEG = 26, LEG_H = 118, OVER = 46;
const TABLE_TOP: Box = { x: -OVER, y: PILE_BASE, z: OVER, w: PILE_W + 2 * OVER, h: TOP_T, d: PILE_W + 2 * OVER };
const PIVOT = { x: PILE_W / 2, y: PILE_BASE, z: -PILE_W / 2 };
const legAt = (x: number, z: number): Box => ({ x, y: PILE_BASE + TOP_T, z, w: LEG, h: LEG_H, d: LEG });
const IN = 22; // retrait des pieds sous le plateau
const LEGS: Box[] = [ // du fond vers l'avant (ordre de dessin)
  legAt(-OVER + IN, -PILE_W - OVER + IN + LEG), legAt(PILE_W + OVER - IN - LEG, -PILE_W - OVER + IN + LEG),
  legAt(-OVER + IN, OVER - IN), legAt(PILE_W + OVER - IN - LEG, OVER - IN),
];

function face(cls: string, w: number, h: number, m: string): HTMLElement {
  const f = document.createElement("div");
  f.className = cls;
  f.style.width = `${w}px`;
  f.style.height = `${h}px`;
  f.style.transform = m;
  return f;
}

function table(): HTMLElement {
  const t = document.createElement("div");
  t.className = "vy-table";
  for (const leg of LEGS) {
    const m = boxMatrices(VIEW, leg, PILE_TURN, PIVOT);
    const g = document.createElement("div");
    g.className = "vy-table-leg";
    g.append(face("vy-table-leg-front", leg.w, leg.h, toCss(m.front)), face("vy-table-leg-side", leg.d, leg.h, toCss(m.side)));
    t.appendChild(g);
  }
  const m = boxMatrices(VIEW, TABLE_TOP, PILE_TURN, PIVOT);
  t.append(
    face("vy-table-top", TABLE_TOP.w, TABLE_TOP.d, toCss(m.top)),
    face("vy-table-side", TABLE_TOP.d, TABLE_TOP.h, toCss(m.side)),
    face("vy-table-front", TABLE_TOP.w, TABLE_TOP.h, toCss(m.front)),
  );
  return t;
}

export function mountRecentPile(root: HTMLElement, deps: { play(e: HistoryEntry): void }): RecentPile {
  let seen = new Set<string>(); // disques de l'affichage précédent
  let first = true;
  let scale = 1;
  let zoom: HTMLElement | null = null;
  // Faces projetées par le calcul (pile3d) : aucune 3D du navigateur, seulement une mise à l'échelle.
  const applyScale = () => {
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
      // Un disque vient d'être posé : celui du dessus n'était pas sur la pile juste avant. (Après une
      // reprise, le nouveau dessus était déjà là, en dessous : il ne bouge pas.)
      const dropped = !first && shown[0] !== undefined && !seen.has(key(shown[0]));
      seen = new Set(shown.map(key));
      first = false;
      root.innerHTML = "";
      const pile = document.createElement("div");
      pile.className = "vy-recent-3d";
      root.appendChild(pile);
      zoom = pile;
      applyScale();
      pile.appendChild(table()); // sous les disques
      let topDisc: HTMLElement | null = null;
      // Du plus ancien (en bas) au plus récent (en haut). Chaque disque est un petit volume :
      // tranche de face, côté droit (visible puisque la pile a pivoté) et pochette posée à plat dessus
      // (visible là où le disque dépasse de ceux qui le recouvrent).
      for (let level = 0; level < n; level++) {
        const it = shown[n - 1 - level]!;
        const { dx, yaw } = jitter(key(it));
        const d = document.createElement("div");
        d.className = "vy-recent-disc";
        d.dataset.level = String(level);
        d.dataset.yaw = String(PILE_TURN + yaw); // la pile a pivoté (PILE_TURN), chaque disque en plus
        d.dataset.dx = String(dx);
        const m = faceMatrices(VIEW, level, dx, PILE_TURN + yaw);
        const b = document.createElement("button");
        b.className = "vy-recent";
        b.dataset.track = it.trackId;
        b.setAttribute("aria-label", `${it.title} — ${it.artist}`);
        b.title = `${it.title} — ${it.artist}`;
        b.innerHTML = `<b>${escapeHtml(it.title)}</b><span>${escapeHtml(it.artist)}</span>`;
        b.style.transform = toCss(m.front);
        const side = document.createElement("div");
        side.className = "vy-recent-side";
        side.style.transform = toCss(m.side);
        // Côté plus sombre, par la couleur.
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
        lid.style.transform = toCss(m.top);
        // Ordre de dessin : pochette, côté, puis tranche (devant).
        d.append(lid, side, b);
        // Tout le disque est cliquable (tranche, côté, pochette) ; au clavier, la tranche suffit.
        d.onclick = () => deps.play(it);
        pile.appendChild(d);
        topDisc = d;
        if (!tints.has(key(it)) && it.artworkUrl) {
          tints.set(key(it), null);
          imageTint(it.artworkUrl).then((t) => { tints.set(key(it), t); if (b.isConnected) paintAll(); });
        }
      }
      if (n === 0) return;
      if (dropped && topDisc) {
        topDisc.dataset.dropped = "1";
        // Posé depuis le disque : il arrive d'en haut à gauche et se dépose sur la pile.
        const from = { opacity: 0, translate: "-140px -70px" };
        const to = { opacity: 1, translate: "0 0" };
        topDisc?.animate?.([from, to], { duration: 650, easing: "cubic-bezier(.2,.75,.2,1)" });
      }
    },
  };
}
