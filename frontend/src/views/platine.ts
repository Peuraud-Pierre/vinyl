import { activeSource, api, onHistory, onSettings, onSnapshot, onSpectrum, onStats, playPause, type Item, type Snapshot, type StatsSnapshot } from "../api";
import { go, type Route } from "../router";
import { openShareCard } from "../share/sharecard";
import { Turntable } from "../turntable/turntable";
import { mountDeck } from "../ui/deck";
import { mountEqualizer } from "../ui/equalizer";
import { icons } from "../ui/icons";
import { mountRecentPile, type RecentPile } from "../ui/recentpile";
import { mountOdometer } from "../ui/odometer";
import { mountRoomTable } from "../ui/roomtable";
import { mountVolume } from "../ui/volume";
import { mountLyricsBoard } from "../ui/lyricsboard";
import { mountFavorite } from "../ui/favorite";
import { mountAddToPlaylist } from "../ui/addtoplaylist";
import { phonePreview, setPhonePreview } from "../ui/phoneframe";
import { mountWallShelf, openPlaylistShelf, SHELF_MAX } from "../ui/wallshelf";
import { estimatePosition } from "../turntable/clock";
import { DECK_IN_STAGE, EQ_IN_STAGE, isPhone, layoutScene } from "./scene";

// Place réservée à droite pour la gélule et les boutons ronds.
const CONTROLS_W = 140;
export function mount(el: HTMLElement): () => void {
  el.innerHTML = `
    <div id="pl-scroll" class="shelf-scroll h-screen overflow-y-auto">
    <section class="relative h-screen">
      <h1 id="pl-title" class="sr-only"></h1>
      <div id="pl-table" aria-hidden="true"></div>
      <div id="pl-wrap" class="absolute" style="left:0;top:0;bottom:0;right:${CONTROLS_W}px">
        <div id="pl-panel" class="vy-plinth" aria-hidden="true"></div>
        <div id="pl-scaled" class="absolute" style="transform-origin:0 0"><div id="tt"></div>
          <div id="pl-deck" class="absolute" style="left:${DECK_IN_STAGE.x}px;top:${DECK_IN_STAGE.y}px"></div>
        </div>
        <div id="pl-eq" class="absolute"></div>
        <div id="pl-vol" class="absolute" style="translate:-50% -50%"></div>
        <div id="pl-lyrics" class="absolute"></div>
        <div id="pl-recent" class="absolute" aria-label="Derniers morceaux écoutés"></div>
        <div id="pl-shelf"></div>
        <p id="pl-recent-msg" class="glass pointer-events-none absolute hidden rounded-full px-5 py-2 text-[14px]" role="status"></p>
      </div>
      <button id="pl-phone" class="glass round-btn vy-drop fixed z-30" style="right:175px;top:28px" aria-label="Aperçu téléphone" title="Aperçu téléphone (développement)" hidden>${icons.phone}</button>
      <button id="pl-share" class="glass round-btn vy-drop fixed z-30" style="right:113px;top:28px" aria-label="Partager">${icons.share}</button>
      <button id="pl-settings" class="glass round-btn vy-drop fixed z-30" style="right:51px;top:28px" aria-label="Réglages">${icons.settings}</button>
      <div id="pb-top" class="vy-ptop">
        <button id="pb-lib" class="glass round-btn" aria-label="Bibliothèque">${icons.library}</button>
        <button id="pb-settings" class="glass round-btn" aria-label="Réglages">${icons.settings}</button>
        <button id="pb-phone" class="glass round-btn" aria-label="Revenir à l'affichage ordinateur" title="Affichage ordinateur (développement)" hidden>${icons.desktop}</button>
      </div>
      <div id="pl-pbar" class="vy-pbar" aria-label="Morceau en cours">
        <div class="vy-pbar-head">
          <div class="vy-pbar-now"><b id="pb-title"></b><span id="pb-artist"></span></div>
          <button id="pb-add" class="vy-pbar-round" hidden></button>
          <button id="pb-fav" class="vy-pbar-round" hidden></button>
          <div id="pb-menu" class="vy-pbar-menu"></div>
        </div>
        <div class="vy-pbar-time"><span id="pb-pos">0:00</span><i><b id="pb-fill"></b></i><span id="pb-left"></span></div>
        <div class="vy-pbar-keys">
          <span class="vy-pbar-key"><button id="pb-pp" class="vy-key" aria-label="Lecture">${icons.play}</button><small id="pb-pp-cap">PLAY</small></span>
          <div id="pb-deck"></div>
        </div>
      </div>
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
  const tt = new Turntable($("tt"), { variant: "stage" });
  // Plus de pochette sur la table : elle est sur le tableau du mur (ui/lyricsboard).
  $("tt").querySelector<HTMLElement>(".tt-sleeve")?.style.setProperty("display", "none");
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
  // Tableau des paroles au mur.
  const lyrics = mountLyricsBoard($("pl-lyrics"), {
    lyrics: (np) => api.Lyrics(np.source, np.trackId, np.title, np.artist, np.album, np.durationMs),
    favorite: { get: (np) => api.IsFavorite(np.source, np.trackId), set: (np, on) => api.SetFavorite(np.source, np.trackId, on) },
    playlist: {
      playlists: (source) => api.EditablePlaylists(source).then((l) => l ?? []),
      add: (np, id) => api.AddToPlaylist(np.source, np.trackId, id),
    },
  });
  // Téléphone : cœur et « + » à côté du titre (le tableau des paroles n'est pas affiché).
  const pbFav = mountFavorite($("pb-fav") as HTMLButtonElement, {
    get: (np) => api.IsFavorite(np.source, np.trackId), set: (np, on) => api.SetFavorite(np.source, np.trackId, on),
  });
  const pbAdd = mountAddToPlaylist($("pb-add") as HTMLButtonElement, $("pb-menu"), {
    playlists: (source) => api.EditablePlaylists(source).then((l) => l ?? []),
    add: (np, id) => api.AddToPlaylist(np.source, np.trackId, id),
  });
  // Volume de l'ordinateur : molette en haut à gauche du socle (pas de bouton).
  const volume = mountVolume($("pl-vol"), { get: () => api.Volume(), set: (v) => api.SetVolume(v) });
  const eq = mountEqualizer($("pl-eq"), { start: () => api.StartSpectrum(), stop: () => api.StopSpectrum(), onSpectrum });
  const canHear = api.SpectrumAvailable().catch(() => false);
  const setEqLive = (on: boolean) => void canHear.then((ok) => eq.setLive(ok && on));
  const fit = () => {
    // Téléphone en portrait : la platine prend toute la largeur (pas de colonne de commandes) ; voir scene.
    // Ses touches (précédent, aléatoire, suivant) passent dans la barre du bas, à côté de « Lecture ».
    const phone = phonePreview() || isPhone(window.innerWidth, window.innerHeight);
    document.documentElement.classList.toggle("vy-phone", phone);
    $("pl-wrap").style.right = phone ? "0px" : `${CONTROLS_W}px`;
    const deckEl = $("pl-deck"), deckHome = phone ? $("pb-deck") : $("pl-scaled");
    if (deckEl.parentElement !== deckHome) deckHome.append(deckEl);
    const wrap = $("pl-wrap").getBoundingClientRect();
    const L = layoutScene(wrap.width, wrap.height, phone);
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
    Object.assign($("pl-panel").style, { left: `${L.panel.x}px`, top: `${L.panel.y}px`, width: `${L.panel.w}px`, height: `${L.panel.h}px` });
    roomTable.setScale(L.k);
    Object.assign($("pl-lyrics").style, { left: `${L.lyrics.x}px`, top: `${L.lyrics.y}px` });
    lyrics.place(L.lyrics.scale);
    Object.assign($("pl-vol").style, { left: `${L.knob.x}px`, top: `${L.knob.y}px` });
    volume.setScale(L.k);
    // Égaliseur hors de la platine mise à l'échelle : dessiné à sa taille d'affichage (net), même place.
    Object.assign($("pl-eq").style, { left: `${L.stage.x + EQ_IN_STAGE.x * L.stage.scale}px`, top: `${L.stage.y + EQ_IN_STAGE.y * L.stage.scale}px` });
    eq.setScale(L.stage.scale);
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
    msgTimer = window.setTimeout(() => m.classList.add("hidden"), 8000);
  };
  const ro = new ResizeObserver(fit);
  ro.observe($("pl-wrap"));
  window.addEventListener("resize", fit);

  // Touches sous la pochette : précédent, aléatoire, suivant (comme sur une platine).
  const deck = mountDeck($("pl-deck"), {
    previous: () => api.Previous(), next: () => api.Next(),
    setShuffle: (on) => api.SetShuffle(on), modes: () => api.Modes(),
  });
  let deckTrack = "";

  // Pile des derniers morceaux écoutés : un clic saute le morceau en cours et joue celui-ci, sans vider la
  // file du lecteur ; la file continue ensuite normalement.
  recent = mountRecentPile($("pl-recent"), {
    // Puis la pile est relue : elle reste juste même si un événement s'est perdu.
    play: (e) => {
      api.PlayFromHistory(e.source, e.trackId)
        .catch((err) => {
          const msg = String(err);
          showRecentMsg(msg.includes("plus dans la file") ? "Ce morceau n'est plus dans la file de lecture"
            : msg.includes("connecte Spotify") ? "Connecte Spotify dans les réglages pour réécouter ce morceau"
            : msg.includes("Accessibilité") ? "Autorise Vinyl dans Réglages Système › Confidentialité et sécurité › Accessibilité"
            : msg.includes("playlist en cours") ? "Morceau introuvable dans la playlist en cours (triée autrement dans Musique ?)"
            : `Lecture impossible : ${msg.replace(/^Error: |osascript: /g, "").slice(-160)}`); // la fin : l'erreur et son code
        })
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

  // Barre du bas (téléphone) : titre, avancement, touches. L'avancement est estimé entre deux sondages.
  const fmt = (ms: number) => {
    const t = Math.max(0, Math.round(ms / 1000));
    return `${Math.floor(t / 60)}:${String(t % 60).padStart(2, "0")}`;
  };
  let barAt = 0;
  let barTimer = 0;
  const tickBar = () => {
    if (!snap || !document.documentElement.classList.contains("vy-phone")) return;
    const np = snap.nowPlaying;
    const pos = estimatePosition(np.positionMs, np.durationMs, np.state === "playing", barAt, performance.now());
    $("pb-pos").textContent = fmt(pos);
    $("pb-left").textContent = np.durationMs > 0 ? `−${fmt(np.durationMs - pos)}` : "";
    $("pb-fill").style.transform = `scaleX(${np.durationMs > 0 ? Math.min(1, pos / np.durationMs) : 0})`;
  };
  const paintBar = (s: Snapshot) => {
    barAt = performance.now();
    const np = s.nowPlaying, playing = !s.problem && np.state === "playing";
    $("pb-title").textContent = s.problem ? "Vinyl" : np.title;
    $("pb-artist").textContent = s.problem ? "" : np.artist;
    $("pb-pp").innerHTML = playing ? icons.pause : icons.play;
    $("pb-pp").setAttribute("aria-label", playing ? "Pause" : "Lecture");
    $("pb-pp-cap").textContent = playing ? "PAUSE" : "PLAY";
    const cur = s.problem || !np.trackId ? null : np;
    pbFav.update(cur);
    pbAdd.update(cur);
    clearInterval(barTimer);
    barTimer = playing ? window.setInterval(tickBar, 500) : 0;
    tickBar();
  };

  const render = (s: Snapshot) => {
    snap = s;
    deck.setEnabled(!s.problem);
    // Aléatoire relu à chaque changement de morceau (il a pu être réglé dans Musique ou Spotify).
    if (!s.problem && s.nowPlaying.trackId !== deckTrack) { deckTrack = s.nowPlaying.trackId; deck.refresh(); }
    tt.update(s);
    paintBar(s);
    lyrics.update(s);
    eq.setPlaying(!s.problem && s.nowPlaying.state === "playing", s.nowPlaying.trackId);
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
  $("pb-settings").onclick = () => go("reglages");
  // Version de développement : bascule de la fenêtre en taille téléphone, et retour (absent des releases).
  api.Version().then((v) => {
    if (v !== "dev") return;
    $("pl-phone").hidden = false;
    $("pb-phone").hidden = false;
  }).catch(() => {});
  // L'aperçu reste en téléphone même si la fenêtre grandit (plein écran) : cadre centré (ui/phoneframe).
  $("pl-phone").onclick = () => { setPhonePreview(true); api.PhonePreview(true).catch(() => {}); };
  $("pb-phone").onclick = () => { setPhonePreview(false); api.PhonePreview(false).catch(() => {}); };
  $("pb-lib").onclick = () => go("bibliotheque", "albums");
  $("pb-pp").onclick = () => { playPause().catch(() => {}); };
  $("pl-share").onclick = () => {
    const np = snap?.nowPlaying;
    if (np && !snap!.problem) openShareCard({ title: np.title, subtitle: np.artist, artworkUrl: np.artworkUrl });
  };

  const offSnap = onSnapshot(render);
  const offSet = onSettings((s) => { tt.setAppearance(s); setEqLive(s.eqAudio); });
  api.GetSnapshot().then(render);
  api.GetSettings().then((s) => { tt.setAppearance(s); setEqLive(s.eqAudio); });

  return () => {
    clearInterval(barTimer);
    window.removeEventListener("resize", fit);
    pbAdd.destroy();
    document.documentElement.classList.remove("vy-phone");
    offStats();
    offSnap();
    window.removeEventListener("focus", onFocus);
    offHistory();
    clearTimeout(msgTimer);
    offSet();
    ro.disconnect();
    tt.destroy();
    eq.destroy();
    volume.destroy();
    lyrics.destroy();
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
