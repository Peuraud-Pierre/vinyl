import { api, onSettings, onSnapshot, onStats, type Snapshot, type StatsSnapshot } from "../api";
import { go } from "../router";
import { openShareCard } from "../share/sharecard";
import { STAGE_H, STAGE_W, Turntable } from "../turntable/turntable";
import { mountDeck } from "../ui/deck";
import { icons } from "../ui/icons";
import { mountOdometer } from "../ui/odometer";

// Place réservée à droite pour la gélule et les boutons ronds.
const CONTROLS_W = 140;

export function mount(el: HTMLElement): () => void {
  el.innerHTML = `
    <div id="pl-scroll" class="shelf-scroll h-screen overflow-y-auto">
    <section class="relative h-screen">
      <h1 id="pl-title" class="sr-only"></h1>
      <div id="pl-wrap" class="absolute" style="left:0;top:0;bottom:0;right:${CONTROLS_W}px">
        <div id="pl-scaled" class="absolute" style="transform-origin:0 0"><div id="tt"></div>
          <div id="pl-deck" class="absolute" style="left:136px;top:604px"></div>
        </div>
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
  const fit = () => {
    const wrap = $("pl-wrap").getBoundingClientRect();
    const k = Math.min(wrap.width / STAGE_W, wrap.height / STAGE_H);
    const s = $("pl-scaled");
    s.style.transform = `scale(${k})`;
    s.style.left = `${(wrap.width - STAGE_W * k) / 2}px`;
    s.style.top = `${(wrap.height - STAGE_H * k) / 2}px`;
  };
  const ro = new ResizeObserver(fit);
  ro.observe($("pl-wrap"));

  // Touches sous la pochette : précédent, aléatoire, suivant (comme sur une platine).
  const deck = mountDeck($("pl-deck"), {
    previous: () => api.Previous(), next: () => api.Next(),
    setShuffle: (on) => api.SetShuffle(on), modes: () => api.Modes(),
  });
  let deckTrack = "";
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
    offSet();
    ro.disconnect();
    tt.destroy();
  };
}
