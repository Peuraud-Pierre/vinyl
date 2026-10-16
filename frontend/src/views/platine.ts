import { activeSource, api, onHistory, onSettings, onSnapshot, onStats, type Item, type Snapshot, type StatsSnapshot } from "../api";
import { go, type Route } from "../router";
import { openShareCard } from "../share/sharecard";
import { Turntable } from "../turntable/turntable";
import { mountDeck } from "../ui/deck";
import { platform } from "../ui/platform";
import { icons } from "../ui/icons";
import { mountRecentPile, type RecentPile } from "../ui/recentpile";
import { mountOdometer } from "../ui/odometer";
import { mountRoomTable } from "../ui/roomtable";
import { mountSleeveStand } from "../ui/sleevestand";
import { mountWallShelf, openPlaylistShelf, SHELF_MAX } from "../ui/wallshelf";
import { layoutScene, SLEEVE_PERSPECTIVE, SLEEVE_YAW, STAND_DEPTH } from "./scene";

// Place réservée à droite pour la gélule et les boutons ronds.
const CONTROLS_W = 140;
export function mount(el: HTMLElement): () => void {
  el.innerHTML = `
    <div id="pl-scroll" class="shelf-scroll h-screen overflow-y-auto">
    <section class="relative h-screen">
      <h1 id="pl-title" class="sr-only"></h1>
      <div id="pl-table" aria-hidden="true"></div>
      <div id="pl-wrap" class="absolute" style="left:0;top:0;bottom:0;right:${CONTROLS_W}px">
        <div id="pl-scaled" class="absolute" style="transform-origin:0 0"><div id="tt"></div>
          <div id="pl-deck" class="absolute" style="left:531px;top:610px"></div>
        </div>
        <div id="pl-stand-side" class="vy-stand-side" aria-hidden="true"></div>
        <div id="pl-stand"></div>
        <div id="pl-recent" class="absolute" aria-label="Derniers morceaux écoutés"></div>
        <div id="pl-shelf"></div>
        <p id="pl-recent-msg" class="glass pointer-events-none absolute hidden rounded-full px-5 py-2 text-[14px]" role="status"></p>
      </div>
      <button id="pl-share" class="glass round-btn vy-drop fixed z-30" style="right:113px;top:28px" aria-label="Partager">${icons.share}</button>
      <button id="pl-settings" class="glass round-btn vy-drop fixed z-30" style="right:51px;top:28px" aria-label="Réglages">${icons.settings}</button>
      <button id="pl-mini" class="vy-drop fixed z-30 h-[58px] w-[58px] overflow-hidden rounded-[16px] bg-cover bg-center" style="left:44px;bottom:32px;box-shadow:0 0 0 2px rgba(255,255,255,.45),0 8px 20px rgba(0,0,0,.35);background-color:#2a2220" aria-label="Mini-platine" title="Mini-platine (⌘⇧M)"></button>
      <div id="pl-problem" class="glass absolute bottom-10 left-1/2 hidden -translate-x-1/2 rounded-[28px] px-7 py-4 text-center text-[15px]" style="margin-left:-${CONTROLS_W / 2}px"></div>
    </section>
    <footer class="relative flex flex-col items-center gap-7 pb-20 pt-16" style="background:#1b1b1b">
      <div id="pl-odo"></div>
      <button id="pl-stats-share" class="flex items-center gap-3 text-[22px] text-white/90 hover:text-white">
        <span id="pl-since">Morceaux lus</span> ${icons.share}
      </button>
    </footer>
    </div>`;
  const $ = (id: string) => el.querySelector<HTMLElement>("#" + id)!;
  platform().then((p) => ($("pl-mini").title = p === "darwin" ? "Mini-platine (⌘⇧M)" : "Mini-platine (Ctrl+Maj+M)"));
  const tt = new Turntable($("tt"), { variant: "stage" });
  let snap: Snapshot | null = null;

  // Bande « Morceaux lus depuis » sous la platine.
  const setCount = mountOdometer($("pl-odo"));
  let statsNow: StatsSnapshot = { count: 0, since: "" };
  const showStats = (st: StatsSnapshot) => {
    statsNow = st;
    setCount(st.count);
    $("pl-since").textContent = st.since ? `Morceaux lus depuis ${st.since}` : "Morceaux lus";
  };
  const offStats = onStats(showStats);
  api.Stats().then(showStats).catch(() => {});
  $("pl-stats-share").onclick = () => openShareCard({
    title: `${statsNow.count} morceaux lus`, subtitle: statsNow.since ? `depuis ${statsNow.since}` : "", artworkUrl: snap?.nowPlaying.artworkUrl ?? "",
  });

  // Mise à l'échelle de la scène dans la place disponible, centrée.
  const roomTable = mountRoomTable($("pl-table"));
  const stand = mountSleeveStand($("pl-stand"));
  const fit = () => {
    const wrap = $("pl-wrap").getBoundingClientRect();
    const L = layoutScene(wrap.width, wrap.height);
    const s = $("pl-scaled");
    s.style.transform = `scale(${L.stage.scale})`;
    s.style.left = `${L.stage.x}px`;
    s.style.top = `${L.stage.y}px`;
    const r = $("pl-recent");
    r.style.left = `${L.pile.x}px`;
    r.style.top = `${L.pile.y}px`;
    r.style.width = `${500 * L.pile.scale}px`;
    r.style.height = `${460 * L.pile.scale}px`;
    recent?.setScale(L.pile.scale);
    const sh = $("pl-shelf");
    sh.style.left = `${L.shelf.x}px`;
    sh.style.top = `${L.shelf.y}px`;
    sh.style.transform = `scale(${L.shelf.scale})`;
    $("pl-table").style.top = `${L.tableTop}px`;
    // Pochette : sortie de la platine, debout sur son support, tournée de trois quarts vers le tourne-disque
    // (coordonnées de la platine, mise à l'échelle : on convertit depuis la zone). Pochette et support
    // pivotent ensemble autour du bas de la pochette, avec la même perspective.
    const P = SLEEVE_PERSPECTIVE * L.k;
    const sl = $("tt").querySelector<HTMLElement>(".tt-sleeve");
    if (sl) {
      const size = L.sleeve.size / L.stage.scale;
      Object.assign(sl.style, {
        left: `${(L.sleeve.x - L.stage.x) / L.stage.scale}px`, top: `${(L.sleeve.y - L.stage.y) / L.stage.scale}px`,
        width: `${size}px`, height: `${size}px`, transformOrigin: "50% 100%",
        transform: `perspective(${P / L.stage.scale}px) rotateY(${SLEEVE_YAW}deg) rotateX(4deg)`,
      });
    }
    // Support « en écoute » sous la pochette (devant elle : son rebord cache le bas de la pochette).
    const pivot = { x: L.sleeve.x + L.sleeve.size / 2, y: L.sleeve.y + L.sleeve.size };
    Object.assign($("pl-stand").style, {
      left: `${L.stand.x}px`, top: `${L.stand.y}px`, width: `${L.stand.w}px`, height: `${L.stand.h}px`,
      transformOrigin: `${pivot.x - L.stand.x}px ${pivot.y - L.stand.y}px`,
      transform: `perspective(${P}px) rotateY(${SLEEVE_YAW}deg)`,
    });
    // Côté droit du socle : part du bord droit vers l'arrière (rotateY 90° autour de ce bord), puis même
    // rotation de trois quarts que la pochette, autour du même point.
    const sideLeft = L.stand.x + L.stand.w, ox = pivot.x - sideLeft, oy = pivot.y - L.stand.y;
    Object.assign($("pl-stand-side").style, {
      left: `${sideLeft}px`, top: `${L.stand.y}px`, width: `${STAND_DEPTH * L.k}px`, height: `${L.stand.h}px`,
      transformOrigin: `${ox}px ${oy}px`,
      transform: `perspective(${P}px) rotateY(${SLEEVE_YAW}deg) translateX(${-ox}px) rotateY(90deg) translateX(${ox}px)`,
    });
    stand.setScale(L.k);
    roomTable.setScale(L.k);
  };
  let recent: RecentPile | null = null;
  // Message bref au-dessus de la pile (morceau introuvable…).
  let msgTimer = 0;
  const showRecentMsg = (text: string) => {
    const m = $("pl-recent-msg"), r = $("pl-recent");
    m.textContent = text;
    m.style.left = r.style.left;
    m.style.top = `${parseFloat(r.style.top) - 10}px`;
    m.classList.remove("hidden");
    clearTimeout(msgTimer);
    msgTimer = window.setTimeout(() => m.classList.add("hidden"), 3500);
  };
  const ro = new ResizeObserver(fit);
  ro.observe($("pl-wrap"));

  // Touches sous la pochette : précédent, aléatoire, suivant (comme sur une platine).
  const deck = mountDeck($("pl-deck"), {
    previous: () => api.Previous(), next: () => api.Next(),
    setShuffle: (on) => api.SetShuffle(on), modes: () => api.Modes(),
  });
  let deckTrack = "";

  // Pile des derniers morceaux écoutés : un clic relance le morceau et retire ceux posés au-dessus.
  recent = mountRecentPile($("pl-recent"), {
    // Puis la pile est relue : elle reste juste même si un événement s'est perdu.
    play: (e) => {
      api.PlayFromHistory(e.source, e.trackId)
        .catch((err) => showRecentMsg(String(err).includes("plus dans la file") ? "Ce morceau n'est plus dans la file de lecture" : "Lecture impossible"))
        .finally(() => api.History().then((l) => recent?.update(l ?? [])).catch(() => {}));
    },
  });
  const shelf = wireShelf($("pl-shelf"), {
    source: activeSource,
    library: (src) => api.Library(src, "playlists", 0).then((p) => (p.items ?? []) as Item[]),
    current: (src) => api.CurrentPlaylist(src),
    artwork: (src, id) => (src === "applemusic" ? api.ItemArtwork(src, id) : Promise.resolve("")),
    onSettings,
    onSnapshot,
  });
  const offHistory = onHistory((l) => recent?.update(l));
  api.History().then((l) => recent?.update(l ?? [])).catch(() => {});
  fit();
  const onFocus = () => { if (!snap?.problem) deck.refresh(); };
  window.addEventListener("focus", onFocus);

  const render = (s: Snapshot) => {
    snap = s;
    deck.setEnabled(!s.problem);
    // Aléatoire relu à chaque changement de morceau (il a pu être réglé dans Musique ou Spotify).
    if (!s.problem && s.nowPlaying.trackId !== deckTrack) { deckTrack = s.nowPlaying.trackId; deck.refresh(); }
    tt.update(s);
    $("pl-mini").style.backgroundImage = s.nowPlaying.artworkUrl ? `url("${s.nowPlaying.artworkUrl.replace(/["\\]/g, "\\$&")}")` : "";
    $("pl-title").textContent = s.problem ? "Vinyl" : `${s.nowPlaying.title} — ${s.nowPlaying.artist}`;
    const p = $("pl-problem");
    p.classList.toggle("hidden", !s.problem);
    if (s.problem === "no-player") p.textContent = "Lance Musique ou Spotify";
    if (s.problem === "permission") {
      const app = s.problemApp === "spotify" ? "Spotify" : "Musique";
      p.innerHTML = `Vinyl n'a pas l'autorisation de piloter ${app}.<br>Réglages Système → Confidentialité et sécurité → Automatisation → Vinyl.<br>
        <button id="open-auto" class="btn-dark mt-3">Ouvrir les réglages</button>`;
      p.querySelector("#open-auto")!.addEventListener("click", () => api.OpenAutomationSettings());
    }
  };

  $("pl-settings").onclick = () => go("reglages");
  $("pl-mini").onclick = () => api.ToggleMiniPlatine().catch(() => {});
  $("pl-share").onclick = () => {
    const np = snap?.nowPlaying;
    if (np && !snap!.problem) openShareCard({ title: np.title, subtitle: np.artist, artworkUrl: np.artworkUrl });
  };

  const offSnap = onSnapshot(render);
  const offSet = onSettings((s) => tt.setAppearance(s));
  api.GetSnapshot().then(render);
  api.GetSettings().then((s) => tt.setAppearance(s));

  return () => {
    offStats();
    offSnap();
    window.removeEventListener("focus", onFocus);
    offHistory();
    clearTimeout(msgTimer);
    offSet();
    ro.disconnect();
    tt.destroy();
    shelf.destroy();
  };
}

// Playlists de l'étagère murale, par source, gardées tant que l'app tourne (relire la bibliothèque Apple
// Music prend du temps) : un nouvel affichage de la platine les montre tout de suite.
const shelfCache = new Map<string, Item[]>();

// Étagère murale branchée sur la source réellement utilisée (activeSource : réglage, ou lecteur actif en
// mode « auto ») : rechargée quand elle change (réglages ou changement de lecteur) ; un clic ouvre la vue
// Étagère des playlists de cette source.
export function wireShelf(el: HTMLElement, deps: {
  source(): Promise<string>;
  library(source: string): Promise<Item[]>;
  current(source: string): Promise<string>;
  artwork?(source: string, id: string): Promise<string>; // pochette absente de la liste (Apple Music)
  nav?: (route: Route, param: string) => void;
  onSettings(cb: (s: { source: string }) => void): () => void;
  onSnapshot(cb: (s: { nowPlaying: { source: string } }) => void): () => void;
  cache?: Map<string, Item[]>;
}): { destroy(): void } {
  const cache = deps.cache ?? shelfCache;
  let lastSource = "";
  let source: Promise<string> = Promise.resolve(""); // source du chargement en cours (une seule lecture)
  const playlists = async (src: string): Promise<Item[]> => {
    const known = cache.get(src);
    if (known) return known;
    const items = (await deps.library(src)).slice(0, SHELF_MAX);
    if (deps.artwork) {
      await Promise.all(items.map(async (it) => {
        if (!it.artworkUrl) it.artworkUrl = await deps.artwork!(src, it.id).catch(() => "");
      }));
    }
    cache.set(src, items);
    return items;
  };
  const shelf = mountWallShelf(el, {
    load: () => {
      source = deps.source();
      return source.then((s) => { lastSource = s; return playlists(s); });
    },
    current: () => source.then((s) => deps.current(s)),
    open: () => void deps.source().then((s) => openPlaylistShelf(s, deps.nav)).catch(() => openPlaylistShelf(lastSource, deps.nav)),
  });
  // Rechargement seulement si la source réellement utilisée a changé.
  const check = () => void deps.source().then((s) => { if (s !== lastSource) shelf.refresh(); }).catch(() => {});
  const offSettings = deps.onSettings(() => check());
  let player = "";
  const offSnapshot = deps.onSnapshot((snap) => {
    const s = snap.nowPlaying.source;
    if (s && s !== player) { player = s; check(); }
  });
  return { destroy() { offSettings(); offSnapshot(); shelf.destroy(); } };
}
