// Pile des derniers morceaux écoutés, à droite du disque : pochettes couchées les unes sur les autres
// après avoir lu le disque, vues de face. On voit leurs tranches (couleur de la pochette, titre),
// du bas vers le haut ; la plus récente est en haut et sa pochette est visible, posée à plat.
import type { HistoryEntry } from "../api";
import { escapeHtml } from "../views/grid";
import { spineColors } from "../views/shelf";
import { faceMatrices, toCss, type PileView } from "./pile3d";
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

// Vue de la pile : un peu d'en haut (point de fuite au-dessus, à 40 % de la largeur).
const VIEW: PileView = { width: PILE_W, height: 460, depth: PILE_W, slab: SLAB, eye: { x: PILE_W * 0.4, y: -70 }, perspective: 1800 };

export function mountRecentPile(root: HTMLElement, deps: { play(e: HistoryEntry): void }): RecentPile {
  let seen = new Set<string>(); // disques de l'affichage précédent
  let busyUntil = 0; // fin de l'envol en cours : la pile mise à jour attend jusque-là
  let pending: HistoryEntry[] | null = null;
  let timer: ReturnType<typeof setTimeout> | null = null;
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
      // Pendant un envol, la nouvelle pile attend la fin de l'animation (sinon, disparition en plein vol).
      const wait = busyUntil - Date.now();
      if (wait > 0) {
        pending = list;
        if (!timer) timer = setTimeout(() => { timer = null; const l = pending!; pending = null; this.update(l); }, wait);
        return;
      }
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
        d.onclick = () => {
          // Ceux posés au-dessus s'envolent vers le haut, celui-ci part vers la platine (à gauche) ;
          // ceux du dessous ne bougent pas. La pile mise à jour arrive ensuite du lecteur.
          busyUntil = Date.now() + 480 + (n - 1 - level) * 45;
          pile.querySelectorAll<HTMLElement>(".vy-recent-disc").forEach((o) => {
            const l = Number(o.dataset.level);
            if (l < level) return;
            o.dataset.leaving = l === level ? "play" : "up";
            o.style.pointerEvents = "none";
            o.querySelectorAll<HTMLElement>("button,div").forEach((f) => (f.style.pointerEvents = "none"));
            const to = l === level ? { opacity: 0, translate: "-260px -60px" } : { opacity: 0, translate: "0 -320px" };
            o.animate?.([{ opacity: 1, translate: "0 0" }, to], {
              duration: 480, delay: (n - 1 - l) * 45, easing: "cubic-bezier(.4,0,.2,1)", fill: "forwards",
            });
          });
          deps.play(it);
        };
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
