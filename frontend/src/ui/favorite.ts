// Cœur du morceau en cours (favori de Musique, titre liké sur Spotify) : relu à chaque changement de
// morceau ; un clic le bascule tout de suite, puis le lecteur confirme (retour en arrière en cas d'échec).
import type { NowPlaying } from "../api";

export interface FavoriteDeps {
  get(np: NowPlaying): Promise<boolean>;
  set(np: NowPlaying, on: boolean): Promise<unknown>;
}

export const HEART = '<svg viewBox="0 0 24 24" width="15" height="15" aria-hidden="true"><path d="M12 20.5s-7.5-4.6-9.2-9.3C1.6 7.9 3.8 4.5 7.2 4.5c2 0 3.6 1.1 4.8 2.8 1.2-1.7 2.8-2.8 4.8-2.8 3.4 0 5.6 3.4 4.4 6.7-1.7 4.7-9.2 9.3-9.2 9.3z"/></svg>';

export function mountFavorite(btn: HTMLButtonElement, deps: FavoriteDeps): { update(np: NowPlaying | null): void } {
  btn.innerHTML = HEART;
  let np: NowPlaying | null = null;
  let key = "";
  let on = false;
  let gen = 0;
  const paint = (v: boolean, title = v ? "Retirer des favoris" : "Ajouter aux favoris") => {
    on = v;
    btn.setAttribute("aria-pressed", String(v));
    btn.setAttribute("aria-label", title);
    btn.title = title;
  };
  paint(false);
  btn.onclick = (e) => {
    e.stopPropagation();
    if (!np) return;
    const want = !on, g = ++gen, cur = np;
    paint(want);
    btn.classList.remove("vy-fav-pop");
    void btn.offsetWidth; // relance l'animation
    if (want) btn.classList.add("vy-fav-pop");
    deps.set(cur, want).catch((err) => { if (g === gen) paint(!want, String(err).replace(/^Error: /, "")); });
  };
  return {
    update(next) {
      np = next;
      btn.hidden = !next;
      const k = next ? `${next.source}|${next.trackId}` : "";
      if (k === key) return;
      key = k;
      const g = ++gen;
      paint(false);
      if (next) deps.get(next).then((v) => { if (g === gen) paint(v); }).catch(() => {});
    },
  };
}
