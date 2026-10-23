// Aperçu téléphone (version de développement seulement) : l'app s'affiche dans un cadre aux proportions
// d'un iPhone 15, centré, quelle que soit la taille de la fenêtre — plein écran compris, la disposition
// téléphone ne change pas. Le choix est retenu d'une ouverture à l'autre.
export const PHONE = { w: 390, h: 844 };
const KEY = "vinyl.phonePreview";
const MARGIN = 24; // autour du cadre quand la fenêtre est plus grande qu'un téléphone

/** Taille du cadre pour une fenêtre : la fenêtre entière si elle a déjà la taille d'un téléphone, sinon le
 *  plus grand cadre aux proportions d'un iPhone qui y tient (avec une marge). */
export function phoneFrame(winW: number, winH: number): { w: number; h: number; framed: boolean } {
  if (winW <= PHONE.w + 2 * MARGIN) return { w: winW, h: winH, framed: false };
  let h = winH - 2 * MARGIN;
  let w = (h * PHONE.w) / PHONE.h;
  if (w > winW - 2 * MARGIN) {
    w = winW - 2 * MARGIN;
    h = (w * PHONE.h) / PHONE.w;
  }
  return { w: Math.round(w), h: Math.round(h), framed: true };
}

let on = false;

/** Aperçu téléphone actif. */
export function phonePreview(): boolean {
  return on;
}

const apply = () => {
  const root = document.documentElement;
  root.classList.toggle("vy-phone-frame", on);
  if (!on) return;
  const f = phoneFrame(window.innerWidth, window.innerHeight);
  root.style.setProperty("--pf-w", `${f.w}px`);
  root.style.setProperty("--pf-h", `${f.h}px`);
  root.classList.toggle("vy-phone-framed", f.framed);
};

/** Active ou coupe l'aperçu (et le retient). */
export function setPhonePreview(v: boolean): void {
  on = v;
  try { localStorage.setItem(KEY, v ? "1" : "0"); } catch { /* stockage indisponible */ }
  apply();
}

/** À l'ouverture d'une version de développement : reprend le choix retenu et suit la taille de la fenêtre. */
export function initPhonePreview(): void {
  try { on = localStorage.getItem(KEY) === "1"; } catch { on = false; }
  window.addEventListener("resize", apply);
  apply();
}
