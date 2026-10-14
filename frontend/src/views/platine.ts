import { api, onSettings, onSnapshot, type Snapshot } from "../api";
import { go } from "../router";
import { openShareCard } from "../share/sharecard";
import { STAGE_H, STAGE_W, Turntable } from "../turntable/turntable";
import { icons } from "../ui/icons";

// Place réservée à droite pour la gélule et les boutons ronds.
const CONTROLS_W = 170;

export function mount(el: HTMLElement): () => void {
  el.innerHTML = `
    <section class="relative h-screen">
      <h1 id="pl-title" class="sr-only"></h1>
      <div id="pl-wrap" class="absolute" style="left:0;top:0;bottom:0;right:${CONTROLS_W}px">
        <div id="pl-scaled" class="absolute" style="transform-origin:0 0"><div id="tt"></div></div>
      </div>
      <button id="pl-share" class="glass round-btn fixed z-30" style="right:142px;top:30px" aria-label="Partager">${icons.share}</button>
      <button id="pl-settings" class="glass round-btn fixed z-30" style="right:60px;top:30px" aria-label="Réglages">${icons.settings}</button>
      <div id="pl-problem" class="glass absolute bottom-10 left-1/2 hidden -translate-x-1/2 rounded-[28px] px-7 py-4 text-center text-[15px]" style="margin-left:-${CONTROLS_W / 2}px"></div>
    </section>`;
  const $ = (id: string) => el.querySelector<HTMLElement>("#" + id)!;
  const tt = new Turntable($("tt"), { variant: "stage" });
  let snap: Snapshot | null = null;

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

  const render = (s: Snapshot) => {
    snap = s;
    tt.update(s);
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
  $("pl-share").onclick = () => {
    const np = snap?.nowPlaying;
    if (np && !snap!.problem) openShareCard({ title: np.title, subtitle: np.artist, artworkUrl: np.artworkUrl });
  };

  const offSnap = onSnapshot(render);
  const offSet = onSettings((s) => tt.setAppearance(s));
  api.GetSnapshot().then(render);
  api.GetSettings().then((s) => tt.setAppearance(s));

  return () => {
    offSnap();
    offSet();
    ro.disconnect();
    tt.destroy();
  };
}
