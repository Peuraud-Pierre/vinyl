// Palette de l'app tirée de la pochette en cours (fond, verre, encre).
export type RGB = [number, number, number];

export const FALLBACK_TINT = "#8a5f57";

export function hexToRgb(hex: string): RGB | null {
  const m = /^#([0-9a-f]{6})$/i.exec(hex.trim());
  if (!m) return null;
  const v = parseInt(m[1]!, 16);
  return [(v >> 16) & 255, (v >> 8) & 255, v & 255];
}

export function rgbToHex([r, g, b]: RGB): string {
  return "#" + [r, g, b].map((c) => Math.round(Math.min(255, Math.max(0, c))).toString(16).padStart(2, "0")).join("");
}

/** Luminance relative (sRGB), de 0 (noir) à 1 (blanc). */
export function luminance([r, g, b]: RGB): number {
  const lin = (c: number) => {
    const s = c / 255;
    return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b);
}

/** Couleur moyenne de pixels RGBA ; les pixels saturés pèsent plus, les transparents sont ignorés. */
export function averageColor(px: Uint8ClampedArray): RGB | null {
  let r = 0, g = 0, b = 0, w = 0;
  for (let i = 0; i + 3 < px.length; i += 4) {
    if (px[i + 3]! < 128) continue;
    const pr = px[i]!, pg = px[i + 1]!, pb = px[i + 2]!;
    const sat = (Math.max(pr, pg, pb) - Math.min(pr, pg, pb)) / 255;
    const weight = 0.15 + sat * sat * 4;
    r += pr * weight; g += pg * weight; b += pb * weight; w += weight;
  }
  return w > 0 ? [r / w, g / w, b / w] : null;
}

function toHsl([r, g, b]: RGB): [number, number, number] {
  const rn = r / 255, gn = g / 255, bn = b / 255;
  const max = Math.max(rn, gn, bn), min = Math.min(rn, gn, bn);
  const l = (max + min) / 2;
  if (max === min) return [0, 0, l];
  const d = max - min;
  const s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
  const h = max === rn ? (gn - bn) / d + (gn < bn ? 6 : 0) : max === gn ? (bn - rn) / d + 2 : (rn - gn) / d + 4;
  return [h * 60, s, l];
}

function fromHsl(h: number, s: number, l: number): RGB {
  const k = (n: number) => (n + h / 30) % 12;
  const a = s * Math.min(l, 1 - l);
  const f = (n: number) => l - a * Math.max(-1, Math.min(k(n) - 3, Math.min(9 - k(n), 1)));
  return [f(0) * 255, f(8) * 255, f(4) * 255];
}

export interface Palette {
  tint: string;   // teinte principale (fond)
  light: boolean; // fond clair → encre sombre
  ink: string;    // texte principal
  inkSoft: string;
  bg: string;     // dégradé du fond (CSS)
  glass: string;  // fond du verre dépoli
}

/** Palette à partir d'une couleur de pochette : teinte adoucie, jamais noir ni blanc pur. */
export function paletteFor(hex: string): Palette {
  const rgb = hexToRgb(hex) ?? hexToRgb(FALLBACK_TINT)!;
  const [h, s, l] = toHsl(rgb);
  const tintRgb = fromHsl(h, Math.min(s, 0.55), Math.min(0.82, Math.max(0.2, l)));
  const tint = rgbToHex(tintRgb);
  const light = luminance(tintRgb) > 0.42;
  const edge = light ? 72 : 42;
  return {
    tint,
    light,
    ink: light ? "#231a14" : "#fbf3ef",
    inkSoft: light ? "rgba(35,26,20,.66)" : "rgba(251,243,239,.7)",
    bg: `radial-gradient(ellipse 70% 80% at 60% 46%, color-mix(in srgb, ${tint} 90%, #fff) 0%, ${tint} 36%, color-mix(in srgb, ${tint} ${edge}%, #070404) 82%, color-mix(in srgb, ${tint} ${edge - 22}%, #030202) 100%)`,
    glass: light ? "rgba(255,255,255,.42)" : `color-mix(in srgb, ${tint} 25%, rgba(255,236,230,.58))`,
  };
}

/** Teinte d'une image (même origine ou CORS) ; null si illisible. */
export function imageTint(url: string): Promise<string | null> {
  if (!url) return Promise.resolve(null);
  return new Promise((resolve) => {
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.onload = () => {
      try {
        const c = document.createElement("canvas");
        c.width = c.height = 24;
        const ctx = c.getContext("2d")!;
        ctx.drawImage(img, 0, 0, 24, 24);
        const avg = averageColor(ctx.getImageData(0, 0, 24, 24).data);
        resolve(avg ? rgbToHex(avg) : null);
      } catch {
        resolve(null); // image d'une autre origine sans CORS
      }
    };
    img.onerror = () => resolve(null);
    img.src = url;
  });
}

/** Applique la palette aux variables CSS de la page. */
export function applyPalette(p: Palette): void {
  const s = document.documentElement.style;
  s.setProperty("--tint", p.tint);
  s.setProperty("--ink", p.ink);
  s.setProperty("--ink-soft", p.inkSoft);
  s.setProperty("--glass", p.glass);
  document.documentElement.dataset.tone = p.light ? "light" : "dark";
}
