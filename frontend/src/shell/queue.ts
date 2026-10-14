// « File d'attente » en plein écran, comme MD Vinyl (vidéo screen/ScreenRecording…, 31–37 s) :
// fond teinté sombre, grande pochette inclinée à gauche, titre et boutons aléatoire / répéter
// en haut, liste du morceau en cours puis des suivants avec l'album en petit à gauche.
import { api, onSnapshot, type QueueState, type Snapshot } from "../api";
import { formatDuration } from "../ui/format";
import { icons } from "../ui/icons";
import { escapeHtml } from "../views/grid";
import { nextRepeat } from "./repeat";

const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;

const row = (album: string, title: string, artist: string, extra: string, current: boolean) => `
  <li class="flex items-start gap-6 py-4">
    <span class="w-36 shrink-0 pt-1 text-[11px] font-semibold uppercase leading-tight tracking-wide opacity-45">${escapeHtml(album)}</span>
    <span class="min-w-0 flex-1">
      <b class="block truncate text-[19px] ${current ? "font-bold" : "font-semibold"}">${current ? "· " : ""}${escapeHtml(title)}</b>
      <span class="block truncate text-[14px] opacity-60">${escapeHtml(artist)}</span>
    </span>
    <span class="pt-1 text-[13px] tabular-nums opacity-50">${extra}</span>
  </li>`;

export interface QueuePanel { toggle(force?: boolean): void; refresh(): void; isOpen(): boolean; onToggle(f: (open: boolean) => void): void; }

export function mountQueuePanel(root: HTMLElement): QueuePanel {
  const panel = document.createElement("section");
  panel.id = "q-panel";
  panel.className = "fixed inset-0 z-[25] hidden overflow-hidden";
  panel.style.cssText = "background:radial-gradient(ellipse 80% 90% at 28% 40%, color-mix(in srgb, var(--tint) 60%, #1a110e) 0%, color-mix(in srgb, var(--tint) 30%, #0b0706) 75%);-webkit-backdrop-filter:blur(30px) saturate(1.3);backdrop-filter:blur(30px) saturate(1.3);color:#f6ece6";
  panel.setAttribute("aria-label", "File d'attente");
  panel.innerHTML = `
    <div id="q-cover" class="absolute bg-cover bg-center" style="left:92px;top:14vh;width:min(38vw,46vh);aspect-ratio:1;transform:rotate(-6deg);background-color:#2a2220;border-radius:4px;box-shadow:0 30px 70px rgba(0,0,0,.55)"></div>
    <div class="absolute flex flex-col" style="left:calc(92px + min(38vw,46vh) + 6vw);right:170px;top:30px;bottom:30px">
      <header class="flex items-center gap-3 py-2">
        <h2 class="flex-1 text-[22px] font-bold">File d'attente</h2>
        <button id="q-shuffle" class="round-btn" aria-label="Lecture aléatoire" aria-pressed="false">${icons.shuffle}</button>
        <button id="q-repeat" class="round-btn" aria-label="Répéter" aria-pressed="false">${icons.repeat}</button>
      </header>
      <ol id="q-list" class="shelf-scroll mt-4 min-h-0 flex-1 overflow-y-auto pr-2"></ol>
      <p id="q-note" class="py-2 text-[13px] opacity-60"></p>
      <div class="flex gap-3 pt-2 opacity-80">
        <button id="q-prev" class="round-btn h-10 w-10" style="background:rgba(255,255,255,.1)" aria-label="Morceau précédent">${icons.prev}</button>
        <button id="q-next" class="round-btn h-10 w-10" style="background:rgba(255,255,255,.1)" aria-label="Morceau suivant">${icons.next}</button>
      </div>
    </div>`;
  root.appendChild(panel);
  const $ = (id: string) => panel.querySelector<HTMLElement>("#" + id)!;
  let state: QueueState | null = null;
  let snap: Snapshot | null = null;
  let lastTrack = "";
  let source = "applemusic";
  const onToggle: Array<(open: boolean) => void> = [];

  const paintToggle = (btn: HTMLElement, on: boolean) => {
    btn.style.background = on ? "#f6ece6" : "rgba(255,255,255,.12)";
    btn.style.color = on ? "#1c1412" : "#f6ece6";
    btn.setAttribute("aria-pressed", String(on));
  };

  const render = () => {
    const np = snap?.nowPlaying;
    const current = np && !snap!.problem ? row(np.album, np.title, np.artist, "", true) : "";
    const up = state?.upcoming ?? [];
    $("q-list").innerHTML = current + up.map((t) => row(t.album, t.title, t.artist, t.durationMs > 0 ? formatDuration(t.durationMs) : "", false)).join("");
    $("q-note").textContent = state?.note || (state && up.length === 0 ? "Rien à suivre" : "");
    if (state) {
      paintToggle($("q-shuffle"), state.shuffle);
      $("q-repeat").innerHTML = state.repeat === "one" ? icons.repeatOne : icons.repeat;
      paintToggle($("q-repeat"), state.repeat !== "off");
    }
  };

  const isOpen = () => !panel.classList.contains("hidden");
  const refresh = () => {
    if (!isOpen()) return;
    api.Queue().then((q) => { state = q; render(); }).catch((e) => {
      state = null;
      render();
      $("q-note").textContent = String(e).includes("aucun lecteur") ? "Lance Musique ou Spotify" : String(e);
    });
  };

  const showNow = (s: Snapshot) => {
    snap = s;
    const np = s.nowPlaying;
    if (np.source) source = np.source;
    $("q-cover").style.backgroundImage = np.artworkUrl ? cssUrl(np.artworkUrl) : "";
    render();
    if (np.trackId !== lastTrack) { lastTrack = np.trackId; refresh(); }
  };

  $("q-prev").onclick = () => api.Previous().catch(() => {});
  $("q-next").onclick = () => api.Next().catch(() => {});
  $("q-shuffle").onclick = async () => { await api.SetShuffle(!(state?.shuffle ?? false)).catch(() => {}); refresh(); };
  $("q-repeat").onclick = async () => { await api.SetRepeat(nextRepeat(state?.repeat ?? "off", source)).catch(() => {}); refresh(); };

  const toggle = (force?: boolean) => {
    const open = force ?? !isOpen();
    if (open === isOpen()) return;
    panel.classList.toggle("hidden", !open);
    if (open) {
      // Arrivée : fond en fondu, pochette qui tombe en place, liste qui glisse.
      panel.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 300, easing: "ease-out" });
      $("q-cover").animate([{ opacity: 0, transform: "translateY(-60px) rotate(-14deg)" }, { opacity: 1, transform: "rotate(-6deg)" }], { duration: 520, easing: "cubic-bezier(.2,.75,.2,1)" });
      $("q-list").animate([{ opacity: 0, transform: "translateX(40px)", filter: "blur(8px)" }, { opacity: 1, transform: "none", filter: "none" }], { duration: 480, delay: 80, easing: "cubic-bezier(.2,.7,.2,1)", fill: "backwards" });
      refresh();
    }
    onToggle.forEach((f) => f(open));
  };
  document.addEventListener("keydown", (e) => { if (e.key === "Escape") toggle(false); });
  window.addEventListener("hashchange", () => toggle(false)); // autre écran : la file se ferme
  onSnapshot(showNow);
  api.GetSnapshot().then(showNow);
  return { toggle, refresh, isOpen, onToggle: (f) => { onToggle.push(f); } };
}
