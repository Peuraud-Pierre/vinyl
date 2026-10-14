// « File d'attente » comme MD Vinyl (vidéo screen/ScreenRecording…, 31–33 s, mesurée image par image) :
// fond teinté habituel, pochette inclinée à gauche (~26 % de la largeur), titre centré en haut,
// colonne étroite au centre (morceau en cours, trait fin, suivants), aléatoire / répéter en haut à
// droite ; l'album n'apparaît qu'en tête de groupe.
import { api, onSnapshot, type QueueState, type Snapshot } from "../api";
import { icons } from "../ui/icons";
import { escapeHtml } from "../views/grid";
import { albumHeaders } from "./queuegroups";
import { nextRepeat } from "./repeat";

const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;

const item = (title: string, artist: string, current: boolean, header: string) => `
  <li class="relative py-3.5">
    ${header ? `<span class="absolute right-full top-4 mr-8 w-40 text-right text-[11px] font-semibold uppercase leading-tight tracking-wide opacity-45">De ${escapeHtml(header)}</span>` : ""}
    <b class="block text-[21px] font-semibold leading-snug">${current ? `<span class="mr-3 opacity-70">··</span>` : ""}${escapeHtml(title)}</b>
    <span class="block truncate text-[17px] opacity-60">${escapeHtml(artist)}</span>
  </li>`;

export interface QueuePanel { toggle(force?: boolean): void; refresh(): void; isOpen(): boolean; onToggle(f: (open: boolean) => void): void; }

export function mountQueuePanel(root: HTMLElement): QueuePanel {
  const panel = document.createElement("section");
  panel.id = "q-panel";
  panel.className = "fixed inset-0 z-[25] hidden overflow-hidden";
  // Même fond teinté que le reste de l'app (version sombre de la teinte de la pochette).
  panel.style.cssText = "color:var(--ink);background:radial-gradient(ellipse 70% 85% at 58% 30%, color-mix(in srgb, var(--tint) 85%, #fff) 0%, var(--tint) 34%, color-mix(in srgb, var(--tint) 45%, #070404) 82%, color-mix(in srgb, var(--tint) 22%, #030202) 100%)";
  panel.setAttribute("aria-label", "File d'attente");
  panel.innerHTML = `
    <div id="q-cover" class="absolute bg-cover bg-center" style="left:4.4vw;top:15vh;width:54vh;max-width:26vw;aspect-ratio:1;transform:rotate(-5deg);background-color:#2a2220;border-radius:3px;box-shadow:0 26px 60px rgba(0,0,0,.5)"></div>
    <h2 class="absolute left-0 right-0 text-center text-[24px] font-medium" style="top:7vh">File d'attente</h2>
    <button id="q-shuffle" class="round-btn glass fixed z-30" style="right:113px;top:28px" aria-label="Lecture aléatoire" aria-pressed="false">${icons.shuffle}</button>
    <button id="q-repeat" class="round-btn glass fixed z-30" style="right:51px;top:28px" aria-label="Répéter" aria-pressed="false">${icons.repeat}</button>
    <div id="q-col" class="shelf-scroll absolute overflow-y-auto" style="left:33.5vw;width:27vw;top:17vh;bottom:6vh">
      <ol id="q-list"></ol>
      <p id="q-note" class="py-3 text-[14px] opacity-60"></p>
    </div>`;
  root.appendChild(panel);
  const $ = (id: string) => panel.querySelector<HTMLElement>("#" + id)!;
  let state: QueueState | null = null;
  let snap: Snapshot | null = null;
  let lastTrack = "";
  let source = "applemusic";
  const listeners: Array<(open: boolean) => void> = [];

  // Bouton actif : noir plein (vidéo : répéter activé) ; inactif : verre.
  const paintToggle = (btn: HTMLElement, on: boolean) => {
    btn.style.background = on ? "#15100e" : "";
    btn.style.color = on ? "#f6ece6" : "";
    btn.setAttribute("aria-pressed", String(on));
  };

  const render = () => {
    const np = snap?.nowPlaying;
    const current = np && !snap!.problem
      ? item(np.title, np.artist, true, "") + `<li aria-hidden="true" class="my-3 border-t" style="border-color:color-mix(in srgb, var(--ink) 22%, transparent)"></li>`
      : "";
    const up = albumHeaders(np?.album ?? "", state?.upcoming ?? []);
    $("q-list").innerHTML = current + up.map(({ track, header }) => item(track.title, track.artist, false, header)).join("");
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

  $("q-shuffle").onclick = async () => { await api.SetShuffle(!(state?.shuffle ?? false)).catch(() => {}); refresh(); };
  $("q-repeat").onclick = async () => { await api.SetRepeat(nextRepeat(state?.repeat ?? "off", source)).catch(() => {}); refresh(); };

  const toggle = (force?: boolean) => {
    const open = force ?? !isOpen();
    if (open === isOpen()) return;
    panel.classList.toggle("hidden", !open);
    if (open) {
      // Arrivée (vidéo, 31 s) : fondu, pochette qui descend en place, liste qui apparaît en flou.
      panel.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 280, easing: "ease-out" });
      $("q-cover").animate([{ opacity: 0, transform: "translateY(-50px) rotate(-12deg)" }, { opacity: 1, transform: "rotate(-5deg)" }], { duration: 520, easing: "cubic-bezier(.2,.75,.2,1)" });
      $("q-col").animate([{ opacity: 0, filter: "blur(10px)" }, { opacity: 1, filter: "blur(0)" }], { duration: 460, delay: 90, easing: "ease-out", fill: "backwards" });
      refresh();
    }
    listeners.forEach((f) => f(open));
  };
  document.addEventListener("keydown", (e) => { if (e.key === "Escape") toggle(false); });
  window.addEventListener("hashchange", () => toggle(false)); // autre écran : la file se ferme
  onSnapshot(showNow);
  api.GetSnapshot().then(showNow);
  return { toggle, refresh, isOpen, onToggle: (f) => { listeners.push(f); } };
}
