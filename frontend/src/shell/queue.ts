// Panneau « File d'attente » (cf. l'enregistrement d'écran de MD Vinyl) : pochette inclinée,
// morceau en cours, titres suivants, boutons aléatoire et répéter.
import { api, onSnapshot, type QueueState, type Snapshot } from "../api";
import { formatDuration } from "../ui/format";
import { icons } from "../ui/icons";
import { escapeHtml } from "../views/grid";
import { nextRepeat } from "./repeat";

const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;

export function mountQueuePanel(root: HTMLElement): { toggle(force?: boolean): void; refresh(): void; isOpen(): boolean } {
  const panel = document.createElement("aside");
  panel.id = "q-panel";
  panel.className = "fixed z-40 hidden flex-col gap-4 overflow-hidden rounded-[34px] p-7";
  panel.style.cssText = "right:130px;top:96px;bottom:28px;width:380px;background:rgba(28,20,18,.72);-webkit-backdrop-filter:blur(28px) saturate(1.4);backdrop-filter:blur(28px) saturate(1.4);color:#f6ece6;box-shadow:0 24px 60px rgba(0,0,0,.45)";
  panel.setAttribute("aria-label", "File d'attente");
  panel.innerHTML = `
    <div id="q-cover" class="mx-auto mt-2 h-44 w-44 shrink-0 rounded-[4px] bg-cover bg-center" style="transform:rotate(-4deg);background-color:#2a2220;box-shadow:0 18px 36px rgba(0,0,0,.5)"></div>
    <h2 class="mt-2 text-center text-[18px] font-bold">File d'attente</h2>
    <div id="q-now" class="border-b border-white/15 pb-3 text-[15px]"></div>
    <ol id="q-list" class="shelf-scroll min-h-0 flex-1 overflow-y-auto"></ol>
    <p id="q-note" class="text-center text-[13px] opacity-70"></p>
    <div class="flex items-center gap-4">
      <button id="q-prev" class="round-btn" style="background:rgba(255,255,255,.12)" aria-label="Morceau précédent">${icons.prev}</button>
      <button id="q-next" class="round-btn" style="background:rgba(255,255,255,.12)" aria-label="Morceau suivant">${icons.next}</button>
      <span class="flex-1"></span>
      <button id="q-shuffle" class="round-btn" aria-label="Lecture aléatoire" aria-pressed="false">${icons.shuffle}</button>
      <button id="q-repeat" class="round-btn" aria-label="Répéter" aria-pressed="false">${icons.repeat}</button>
    </div>`;
  root.appendChild(panel);
  const $ = (id: string) => panel.querySelector<HTMLElement>("#" + id)!;
  let state: QueueState | null = null;
  let lastTrack = "";
  let source = "applemusic";

  const paintToggle = (btn: HTMLElement, on: boolean) => {
    btn.style.background = on ? "#f6ece6" : "rgba(255,255,255,.12)";
    btn.style.color = on ? "#1c1412" : "#f6ece6";
    btn.setAttribute("aria-pressed", String(on));
  };

  const render = (q: QueueState) => {
    state = q;
    const up = q.upcoming ?? [];
    $("q-list").innerHTML = up.map((t) => `
      <li class="flex items-baseline gap-3 py-2.5">
        <span class="min-w-0 flex-1"><b class="block truncate font-semibold">${escapeHtml(t.title)}</b><span class="block truncate text-[13px] opacity-70">${escapeHtml(t.artist)}</span></span>
        <span class="text-[13px] tabular-nums opacity-60">${formatDuration(t.durationMs)}</span>
      </li>`).join("");
    $("q-note").textContent = q.note || (up.length === 0 ? "Rien à suivre" : "");
    paintToggle($("q-shuffle"), q.shuffle);
    $("q-repeat").innerHTML = q.repeat === "one" ? icons.repeatOne : icons.repeat;
    paintToggle($("q-repeat"), q.repeat !== "off");
  };

  const isOpen = () => !panel.classList.contains("hidden");
  const refresh = () => {
    if (!isOpen()) return;
    api.Queue().then(render).catch((e) => { $("q-note").textContent = String(e).includes("aucun lecteur") ? "Lance Musique ou Spotify" : String(e); });
  };

  const showNow = (s: Snapshot) => {
    const np = s.nowPlaying;
    if (np.source) source = np.source;
    $("q-cover").style.backgroundImage = np.artworkUrl ? cssUrl(np.artworkUrl) : "";
    $("q-now").innerHTML = s.problem ? "" : `<b class="block truncate">·· ${escapeHtml(np.title)}</b><span class="block truncate text-[13px] opacity-70">${escapeHtml(np.artist)}</span>`;
    if (np.trackId !== lastTrack) { lastTrack = np.trackId; refresh(); }
  };

  $("q-prev").onclick = () => api.Previous().catch(() => {});
  $("q-next").onclick = () => api.Next().catch(() => {});
  $("q-shuffle").onclick = async () => { await api.SetShuffle(!(state?.shuffle ?? false)).catch(() => {}); refresh(); };
  $("q-repeat").onclick = async () => { await api.SetRepeat(nextRepeat(state?.repeat ?? "off", source)).catch(() => {}); refresh(); };

  const toggle = (force?: boolean) => {
    const open = force ?? !isOpen();
    panel.classList.toggle("hidden", !open);
    panel.classList.toggle("flex", open);
    if (open) {
      panel.classList.remove("vy-slide");
      void panel.offsetWidth; // relance l'animation d'arrivée
      panel.classList.add("vy-slide");
      refresh();
    }
  };
  document.addEventListener("keydown", (e) => { if (e.key === "Escape") toggle(false); });
  document.addEventListener("click", (e) => {
    const t = e.target;
    if (isOpen() && t instanceof Element && !t.closest("#q-panel") && !t.closest("#sh-queue")) toggle(false);
  });
  onSnapshot(showNow);
  api.GetSnapshot().then(showNow);
  return { toggle, refresh, isOpen };
}
