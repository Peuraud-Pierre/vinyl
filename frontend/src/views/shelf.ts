// Logique pure de l'étagère : recherche et couleurs des tranches.
import type { Item } from "../api";
import { hexToRgb, luminance, rgbToHex } from "../ui/theme";

export const normalize = (s: string) => s.normalize("NFD").replace(/[\u0300-\u036f]/g, "").toLowerCase();

export function filterItems(items: Item[], query: string): Item[] {
  const q = normalize(query.trim());
  if (!q) return items;
  return items.filter((i) => normalize(`${i.title} ${i.subtitle}`).includes(q));
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
