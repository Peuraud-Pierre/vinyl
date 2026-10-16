// Cadre commun : gélule de commandes, menu « … », fond teinté par la pochette.
import { api, onSnapshot, playPause, type Snapshot } from "../api";
import { go } from "../router";
import { icons } from "../ui/icons";
import { applyPalette, imageTint, paletteFor } from "../ui/theme";
import { escapeHtml } from "../views/grid";
import { mountQueuePanel } from "./queue";

const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;

export function mountShell(root: HTMLElement, accent: () => string): { refreshTheme: () => void } {
  root.innerHTML = `
    <nav aria-label="Commandes" class="glass fixed z-30 flex flex-col items-center justify-between rounded-[35px] py-[12px]" style="right:40px;top:108px;width:70px;height:262px">
      <button id="sh-lib" class="icon-btn h-[46px] w-[46px]" aria-label="Bibliothèque">${icons.library}</button>
      <div class="flex h-[116px] w-[54px] flex-col items-center justify-around rounded-[27px]" style="background:rgba(255,255,255,.28)">
        <button id="sh-cover" class="h-[34px] w-[34px] cursor-pointer rounded-full bg-cover bg-center" aria-label="Revenir à la platine" style="box-shadow:0 0 0 2px rgba(255,255,255,.6);background-color:var(--accent)"></button>
        <button id="sh-pp" class="icon-btn h-10 w-10" aria-label="Lecture">${icons.play}</button>
      </div>
      <button id="sh-queue" class="icon-btn h-[46px] w-[46px]" aria-label="File d'attente" aria-expanded="false">${icons.queue}</button>
    </nav>
    <button id="sh-more" class="glass round-btn fixed z-30" style="right:51px;bottom:32px" aria-label="Plus" aria-expanded="false">${icons.more}</button>
    <div id="sh-more-pop" class="glass fixed z-30 hidden w-64 rounded-[30px] p-2" style="right:51px;bottom:92px">
      <button id="sh-prev" class="menu-item">${icons.prev} Morceau précédent</button>
      <button id="sh-next" class="menu-item">${icons.next} Morceau suivant</button>
      <button data-go="personnaliser" class="menu-item">${icons.palette} Personnaliser</button>
      <button data-go="reglages" class="menu-item">${icons.settings.replace('width="28" height="28"', 'width="20" height="20"')} Réglages</button>
      <button id="sh-artist" class="menu-item">${icons.person} <span>Page de l'artiste</span></button>
    </div>`;
  const $ = (id: string) => root.querySelector<HTMLElement>("#" + id)!;
  const queue = mountQueuePanel(root);
  let snap: Snapshot | null = null;

  const toggle = (btn: string, pop: string, force?: boolean) => {
    const open = force ?? $(pop).classList.contains("hidden");
    $(pop).classList.toggle("hidden", !open);
    $(btn).setAttribute("aria-expanded", String(open));
  };
  const closeAll = () => toggle("sh-more", "sh-more-pop", false);

  $("sh-lib").onclick = () => go("bibliotheque", "albums");
  $("sh-cover").onclick = () => go("platine");
  $("sh-pp").onclick = () => playPause().catch(() => {});
  $("sh-queue").onclick = (e) => {
    e.stopPropagation();
    closeAll();
    queue.toggle();
  };
  queue.onToggle((open) => $("sh-queue").setAttribute("aria-expanded", String(open)));
  $("sh-prev").onclick = () => { closeAll(); api.Previous().catch(() => {}); };
  $("sh-next").onclick = () => { closeAll(); api.Next().catch(() => {}); };
  $("sh-more").onclick = (e) => { e.stopPropagation(); queue.toggle(false); toggle("sh-more", "sh-more-pop"); };
  $("sh-artist").onclick = () => { if (snap?.nowPlaying.artist) go("artiste", snap.nowPlaying.artist); };
  root.querySelectorAll<HTMLElement>("[data-go]").forEach((b) => (b.onclick = () => go(b.dataset.go as "personnaliser" | "reglages")));
  document.addEventListener("click", (e) => { if (!(e.target instanceof Element && e.target.closest("#sh-more-pop"))) closeAll(); });

  // Fond : deux calques en fondu enchaîné.
  const layers = [document.getElementById("bg-a")!, document.getElementById("bg-b")!];
  let front = 0;
  let themedFor: string | null = null;
  const setTheme = async (art: string, force = false) => {
    if (art === themedFor && !force) return;
    themedFor = art;
    const tint = (await imageTint(art)) ?? accent();
    if (themedFor !== art) return;
    const p = paletteFor(tint);
    applyPalette(p);
    const back = layers[1 - front]!;
    back.style.background = p.bg;
    back.style.opacity = "1";
    layers[front]!.style.opacity = "0";
    front = 1 - front;
  };

  const render = (s: Snapshot) => {
    snap = s;
    const np = s.nowPlaying;
    const playing = !s.problem && np.state === "playing";
    $("sh-pp").innerHTML = playing ? icons.pause : icons.play;
    $("sh-pp").setAttribute("aria-label", playing ? "Pause" : "Lecture");
    $("sh-cover").style.backgroundImage = np.artworkUrl ? cssUrl(np.artworkUrl) : "";
    $("sh-artist").querySelector("span")!.innerHTML = np.artist ? `Page de ${escapeHtml(np.artist)}` : "Page de l'artiste";
    setTheme(s.problem ? "" : np.artworkUrl);
  };
  onSnapshot(render);
  api.GetSnapshot().then(render);
  setTheme("", true);
  return { refreshTheme: () => setTheme(themedFor ?? "", true) };
}
