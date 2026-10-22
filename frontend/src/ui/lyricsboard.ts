// Tableau au mur, entre la platine et l'étagère : cadre fin en graphite, fond d'ardoise, titre du
// morceau en tête, paroles du morceau en cours et fine barre d'avancement en bas. Synchronisées, la
// ligne chantée est éclairée (couleur d'accent) et le texte défile
// pour la garder au milieu ; en texte simple, il défile au fil du morceau. Paroles cherchées par le
// backend (LRCLIB, sinon celles enregistrées dans Musique). Par-dessus, la pochette du morceau (avec titre
// et artiste dessous, comme une affiche) occupe le cadre : un clic la fait remonter et découvre les paroles ; le bouton × la fait redescendre du haut
// (choix retenu d'une fois sur l'autre). Pochette baissée, aucune parole n'est cherchée.
import type { NowPlaying, Snapshot } from "../api";
import { estimatePosition } from "../turntable/clock";
import { escapeHtml } from "../views/grid";
import { mountFavorite, type FavoriteDeps } from "./favorite";

export const LYRICS_W = 330; // px de scène
export const LYRICS_H = 420;

export interface LyricLine { timeMs: number; text: string; }
export interface LyricsData { synced: LyricLine[] | null; plain: string; instrumental: boolean; }

/** Ligne en cours à une position (ms) : la dernière commencée, -1 avant la première. */
export function currentLine(lines: LyricLine[], posMs: number): number {
  let lo = 0, hi = lines.length - 1, found = -1;
  while (lo <= hi) {
    const mid = (lo + hi) >> 1;
    if (lines[mid]!.timeMs <= posMs) { found = mid; lo = mid + 1; } else hi = mid - 1;
  }
  return found;
}

/** Part du texte simple à faire défiler (0 à 1) : on attend un peu au début, on finit un peu avant la fin. */
export function plainProgress(posMs: number, durationMs: number): number {
  if (durationMs <= 0) return 0;
  return Math.min(1, Math.max(0, (posMs / durationMs - 0.06) / 0.86));
}

export interface LyricsBoardDeps { lyrics(np: NowPlaying): Promise<LyricsData>; favorite?: FavoriteDeps; }

const readOpen = () => { try { return localStorage.getItem("vinyl.lyrics") === "open"; } catch { return false; } };
const saveOpen = (open: boolean) => { try { localStorage.setItem("vinyl.lyrics", open ? "open" : "closed"); } catch { /* stockage indisponible */ } };

export interface LyricsBoard {
  update(s: Snapshot): void;
  /** Échelle de la scène. */
  place(k: number): void;
  destroy(): void;
}

export function mountLyricsBoard(root: HTMLElement, deps: LyricsBoardDeps): LyricsBoard {
  root.innerHTML = `
    <div class="vy-lyr" aria-live="off">
      <button class="vy-lyr-fav" hidden></button>
      <button class="vy-lyr-close" aria-label="Cacher les paroles" title="Cacher les paroles">×</button>
      <div class="vy-lyr-slate">
        <p class="vy-lyr-head"><b></b><span></span></p>
        <div class="vy-lyr-view">
          <div class="vy-lyr-text"></div>
          <p class="vy-lyr-note"></p>
        </div>
        <i class="vy-lyr-bar"></i>
        <button class="vy-lyr-cover" aria-label="Voir les paroles" title="Voir les paroles">
          <i class="vy-lyr-cover-bg"></i><i class="vy-lyr-cover-art"></i>
          <span class="vy-lyr-cover-cap"><b></b><span></span></span>
          <span class="vy-lyr-cover-hint">♪ Paroles</span>
        </button>
      </div>
      <span class="vy-lyr-plate">Paroles</span>
    </div>`;
  const box = root.querySelector<HTMLElement>(".vy-lyr")!;
  const view = root.querySelector<HTMLElement>(".vy-lyr-view")!;
  const bar = root.querySelector<HTMLElement>(".vy-lyr-bar")!;
  const text = root.querySelector<HTMLElement>(".vy-lyr-text")!;
  const note = root.querySelector<HTMLElement>(".vy-lyr-note")!;
  const cover = root.querySelector<HTMLElement>(".vy-lyr-cover")!;
  const closeBtn = root.querySelector<HTMLElement>(".vy-lyr-close")!;
  // Cœur du morceau en cours, en haut à gauche (pochette baissée ou paroles).
  const fav = deps.favorite ? mountFavorite(root.querySelector<HTMLButtonElement>(".vy-lyr-fav")!, deps.favorite) : null;
  let artUrl: string | null = null;
  let open = readOpen();
  let wanted: NowPlaying | null = null; // morceau dont il faudra chercher les paroles à l'ouverture
  let snap: Snapshot | null = null;
  let receivedAt = 0;
  let trackKey = "";
  let data: LyricsData | null = null;
  let rows: HTMLElement[] = [];
  let shown = -2;
  let timer = 0;
  let gen = 0;

  const say = (msg: string) => { text.innerHTML = ""; rows = []; note.textContent = msg; };
  say("♪");
  const position = () => {
    const np = snap!.nowPlaying;
    return estimatePosition(np.positionMs, np.durationMs, np.state === "playing", receivedAt, performance.now());
  };
  // Défile pour amener la ligne i au milieu de l'ardoise.
  const center = (i: number) => {
    const row = rows[Math.max(0, i)];
    if (!row) return;
    const y = row.offsetTop + row.offsetHeight / 2 - view.clientHeight / 2;
    text.style.transform = `translateY(${-Math.max(0, y)}px)`;
  };
  let plainMax = -1; // défilement maximal du texte simple (mesuré une fois par mise en page)
  const tick = () => {
    if (!snap || !open) return;
    const np = snap.nowPlaying;
    bar.style.transform = `scaleX(${np.durationMs > 0 ? Math.min(1, position() / np.durationMs) : 0})`;
    if (!data) return;
    if (data.synced?.length) {
      const i = currentLine(data.synced, position() + 250); // un rien d'avance : la ligne s'éclaire au chant
      if (i !== shown) {
        rows[shown]?.classList.remove("on");
        rows[i]?.classList.add("on");
        shown = i;
        center(i);
      }
    } else if (rows.length) {
      if (plainMax < 0) plainMax = Math.max(0, text.scrollHeight - view.clientHeight);
      text.style.transform = `translateY(${-plainProgress(position(), snap.nowPlaying.durationMs) * plainMax}px)`;
    }
  };
  // Minuterie : 200 ms pour suivre les lignes synchronisées, 1 s sinon (texte simple, transition de 1 s ; barre) ;
  // aucune quand le tableau est rangé, le morceau en pause ou la fenêtre cachée.
  let period = 0;
  const run = () => {
    const want = open && !document.hidden && snap?.nowPlaying.state === "playing"
      ? (data?.synced?.length && rows.length ? 200 : 1000) : 0;
    if (want !== period) {
      clearInterval(timer);
      period = want;
      timer = want ? window.setInterval(tick, want) : 0;
    }
    tick();
  };
  const onVisibility = () => run();
  document.addEventListener("visibilitychange", onVisibility);
  const show = (d: LyricsData) => {
    data = d;
    shown = -2;
    text.style.transform = "";
    const lines = d.synced?.length ? d.synced.map((l) => l.text) : d.plain.split("\n");
    if (d.instrumental && !d.synced?.length && !d.plain.trim()) return say("Instrumental");
    if (!lines.some((l) => l.trim())) return say("Pas de paroles pour ce morceau");
    note.textContent = "";
    text.classList.toggle("vy-lyr-plain", !d.synced?.length);
    text.innerHTML = lines.map((l) => `<p>${l.trim() ? escapeHtml(l) : "♪"}</p>`).join("");
    rows = Array.from(text.children) as HTMLElement[];
    plainMax = -1;
    run();
  };
  const load = (np: NowPlaying) => {
    if (!open) { wanted = np; return; } // pochette baissée : rien à chercher pour l'instant
    wanted = null;
    const g = ++gen;
    data = null;
    say("…");
    deps.lyrics(np).then((d) => { if (g === gen) show(d); }).catch(() => { if (g === gen) say("Paroles indisponibles"); });
  };

  // Paroles visibles : la pochette est remontée (cachée en haut du cadre). Sinon elle couvre l'ardoise.
  const paintOpen = () => {
    box.classList.toggle("vy-lyr-open", open);
    cover.tabIndex = open ? -1 : 0;
    cover.setAttribute("aria-hidden", String(open));
    closeBtn.tabIndex = open ? 0 : -1;
  };
  const paintCover = (url: string) => {
    if (url === artUrl) return;
    artUrl = url;
    const css = url ? `url("${url.replace(/["\\]/g, "\\$&")}")` : "";
    cover.querySelectorAll<HTMLElement>("i").forEach((el) => (el.style.backgroundImage = css));
    cover.classList.toggle("vy-lyr-cover-empty", !url);
  };
  // Titre et artiste : en tête de l'ardoise et sous la pochette.
  let caption = "";
  const paintTitle = (title: string, artist: string) => {
    if (title + "\n" + artist === caption) return;
    caption = title + "\n" + artist;
    root.querySelectorAll(".vy-lyr-head, .vy-lyr-cover-cap").forEach((el) => {
      el.querySelector("b")!.textContent = title;
      el.querySelector("span")!.textContent = artist;
    });
  };
  closeBtn.onclick = (e) => {
    e.stopPropagation();
    open = false;
    saveOpen(false);
    paintOpen();
    run();
  };
  cover.onclick = () => {
    open = true;
    saveOpen(true);
    paintOpen();
    if (wanted) load(wanted);
    shown = -2;
    run();
  };
  paintOpen();

  return {
    update(s) {
      snap = s;
      receivedAt = performance.now();
      const np = s.nowPlaying;
      paintCover(s.problem ? "" : np.artworkUrl);
      paintTitle(s.problem ? "" : np.title, s.problem ? "" : np.artist);
      fav?.update(s.problem || !np.trackId ? null : np);
      const key = s.problem || !np.title ? "" : `${np.source}|${np.trackId}|${np.title}`;
      if (key !== trackKey) {
        trackKey = key;
        if (key) load(np);
        else { gen++; data = null; say("♪"); }
      }
      run();
    },
    // zoom comme l'égaliseur : dessiné à sa taille d'affichage, donc net.
    place(k) {
      box.style.zoom = String(k);
      shown = -2;
      plainMax = -1;
      tick();
    },
    destroy() { clearInterval(timer); gen++; document.removeEventListener("visibilitychange", onVisibility); },
  };
}
