export const CARD_W = 1080;
export const CARD_H = 1350;

// fitText tronque le texte avec « … » pour qu'il tienne dans maxWidth (recherche dichotomique).
export function fitText(measure: (s: string) => number, text: string, maxWidth: number): string {
  if (measure(text) <= maxWidth) return text;
  let lo = 0;
  let hi = text.length;
  while (lo < hi) {
    const mid = Math.ceil((lo + hi) / 2);
    if (measure(text.slice(0, mid) + "…") <= maxWidth) lo = mid;
    else hi = mid - 1;
  }
  return text.slice(0, lo).trimEnd() + "…";
}
