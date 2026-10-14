import { api, onSettings, onSnapshot, type Snapshot } from "../api";
import { go } from "../router";
import { openShareCard } from "../share/sharecard";
import { estimatePosition } from "../turntable/clock";
import { Turntable } from "../turntable/turntable";

const fmt = (ms: number) => {
  const s = Math.max(0, Math.floor(ms / 1000));
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
};

export function mount(el: HTMLElement): () => void {
  el.innerHTML = `
    <div class="flex min-h-full flex-col items-center justify-center gap-5 p-8" style="background:radial-gradient(circle at 50% 40%, color-mix(in srgb, var(--accent) 25%, #141414), #141414 70%)">
      <div id="tt"></div>
      <div id="problem" class="hidden text-center text-sm text-neutral-300"></div>
      <div class="text-center">
        <p id="title" class="text-lg font-semibold"></p>
        <button id="artist" class="text-neutral-400 hover:underline"></button>
      </div>
      <div class="flex w-[420px] items-center gap-3 text-xs text-neutral-500">
        <span id="pos">0:00</span>
        <div class="h-1 flex-1 rounded bg-neutral-700"><div id="bar" class="h-full rounded" style="background:var(--accent);width:0"></div></div>
        <span id="dur">0:00</span>
      </div>
      <div class="flex gap-8 text-2xl">
        <button id="prev" aria-label="Précédent">⏮</button>
        <button id="pp" aria-label="Lecture / pause">⏯</button>
        <button id="next" aria-label="Suivant">⏭</button>
      </div>
      <button id="share" class="text-xs text-neutral-400 hover:text-neutral-200">Partager</button>
    </div>`;
  const $ = (id: string) => el.querySelector<HTMLElement>("#" + id)!;
  const tt = new Turntable($("tt"), { variant: "full" });
  let snap: Snapshot | null = null;
  let at = performance.now();

  const render = (s: Snapshot) => {
    snap = s;
    at = performance.now();
    tt.update(s);
    const np = s.nowPlaying;
    $("title").textContent = s.problem ? "" : np.title;
    $("artist").textContent = s.problem ? "" : [np.artist, np.album].filter(Boolean).join(" — ");
    $("dur").textContent = fmt(np.durationMs);
    const p = $("problem");
    p.classList.toggle("hidden", !s.problem);
    if (s.problem === "no-player") p.textContent = "Lance Musique ou Spotify";
    if (s.problem === "permission") {
      const app = s.problemApp === "spotify" ? "Spotify" : "Musique";
      p.innerHTML = `Vinyl n'a pas l'autorisation de piloter ${app}.<br>Réglages Système → Confidentialité et sécurité → Automatisation → Vinyl.<br>
        <button id="open-auto" class="mt-2 rounded px-3 py-1 text-neutral-900" style="background:var(--accent)">Ouvrir les réglages</button>`;
      p.querySelector("#open-auto")!.addEventListener("click", () => api.OpenAutomationSettings());
    }
  };

  const tick = window.setInterval(() => {
    if (!snap) return;
    const np = snap.nowPlaying;
    const pos = estimatePosition(np.positionMs, np.durationMs, np.state === "playing", at, performance.now());
    $("pos").textContent = fmt(pos);
    $("bar").style.width = np.durationMs > 0 ? `${(pos / np.durationMs) * 100}%` : "0";
  }, 250);

  $("prev").onclick = () => api.Previous();
  $("pp").onclick = () => api.PlayPause();
  $("next").onclick = () => api.Next();
  $("share").onclick = () => {
    const np = snap?.nowPlaying;
    if (np && !snap!.problem) openShareCard({ title: np.title, subtitle: np.artist, artworkUrl: np.artworkUrl });
  };
  $("artist").onclick = () => {
    if (snap?.nowPlaying.artist) go("artiste", snap.nowPlaying.artist);
  };

  const offSnap = onSnapshot(render);
  const offSet = onSettings((s) => tt.setAppearance(s));
  api.GetSnapshot().then(render);
  api.GetSettings().then((s) => tt.setAppearance(s));

  return () => {
    offSnap();
    offSet();
    clearInterval(tick);
    tt.destroy();
  };
}
