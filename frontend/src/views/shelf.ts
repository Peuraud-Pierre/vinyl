// Logique pure de l'étagère : recherche et couleurs des tranches.
import type { Item, Track } from "../api";
import { hexToRgb, luminance, rgbToHex } from "../ui/theme";

export const normalize = (s: string) => s.normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLowerCase();

export function filterItems(items: Item[], query: string): Item[] {
  const q = normalize(query.trim());
  if (!q) return items;
  return items.filter((i) => normalize(`${i.title} ${i.subtitle}`).includes(q));
}

/** Positions des titres d'une playlist qui correspondent à la recherche (titre, artiste ou album). */
export function matchingTracks(tracks: Track[], query: string): number[] {
  const q = normalize(query.trim());
  return tracks.flatMap((t, i) => (!q || normalize(`${t.title} ${t.artist} ${t.album}`).includes(q) ? [i] : []));
}

function hashHue(s: string): number {
  let h = 0;
  for (const c of s) h = (h * 31 + c.codePointAt(0)!) >>> 0;
  return h % 360;
}

/** Couleur de la tranche : celle de la pochette si connue, sinon une teinte stable tirée du titre. */
export function spineColors(title: string, coverHex?: string): { bg: string; ink: string } {
  let rgb = coverHex ? hexToRgb(coverHex) : null;
  if (!rgb) {
    const h = hashHue(title) / 360, s = 0.32, l = 0.3;
    const q = l < 0.5 ? l * (1 + s) : l + s - l * s, p = 2 * l - q;
    const f = (t: number) => {
      t = (t + 1) % 1;
      return 255 * (t < 1 / 6 ? p + (q - p) * 6 * t : t < 1 / 2 ? q : t < 2 / 3 ? p + (q - p) * (2 / 3 - t) * 6 : p);
    };
    rgb = [f(h + 1 / 3), f(h), f(h - 1 / 3)];
  }
  return { bg: rgbToHex(rgb), ink: luminance(rgb) > 0.3 ? "#1b1513" : "#f6efe9" };
}

/** Attend p (réussite ou échec) au plus ms millisecondes ; true si p s'est terminé à temps. */
export function settleWithin(p: Promise<unknown>, ms: number): Promise<boolean> {
  return new Promise((resolve) => {
    const timer = setTimeout(() => resolve(false), ms);
    const done = () => { clearTimeout(timer); resolve(true); };
    p.then(done, done);
  });
}

// Vue pile : pochettes de PILE_SIZE px empilées, chacune dépassant de PILE_STEP px (cf. .vy-pile).
export const PILE_SIZE = 300;
export const PILE_STEP = 56;

/** Marge haute et basse de la pile : de quoi amener le premier et le dernier élément au milieu. */
export function pilePad(viewHeight: number): number {
  return Math.max(0, viewHeight / 2 - PILE_SIZE / 2);
}

/** Élément dont la pochette est au milieu de l'écran (-1 si la liste est vide). */
export function pileCurrent(scrollTop: number, viewHeight: number, pad: number, count: number): number {
  if (count <= 0) return -1;
  const i = Math.round((scrollTop + viewHeight / 2 - pad - PILE_SIZE / 2) / PILE_STEP);
  return Math.min(count - 1, Math.max(0, i));
}

/** Message quand la bibliothèque Apple Music n'est pas accessible (Windows : jamais). */
export function unavailableLibraryMessage(platform: string): string {
  return platform === "windows" ? "La bibliothèque Apple Music n'est disponible que sur Mac." : "Non disponible pour Apple Music.";
}
