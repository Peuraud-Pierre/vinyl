// Page album / playlist façon MD Vinyl (cf. screen/playlistalbum*.png) : pochette avec reflet,
// infos centrées, pilules « Lecture » / « Lecture aléatoire », liste des titres avec durées et cœurs.
import { api, type Collection } from "../api";
import { go, parseAlbumParam } from "../router";
import { openShareCard } from "../share/sharecard";
import { formatDuration } from "../ui/format";
import { icons } from "../ui/icons";
import { imageTint, paletteFor } from "../ui/theme";
import { escapeHtml } from "./grid";

const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;

export function mount(el: HTMLElement, param: string): () => void {
  const { source, id } = parseAlbumParam(param);
  let alive = true;
  el.innerHTML = `
    <section id="al-root" class="relative h-screen overflow-hidden" style="color:var(--ink)">
      <button id="al-back" class="glass round-btn vy-drop fixed z-30" style="left:92px;top:28px" aria-label="Retour">${icons.back}</button>
      <button id="al-share" class="glass round-btn vy-drop fixed z-30" style="right:51px;top:28px" aria-label="Partager">${icons.share}</button>
      <h1 id="al-bar" class="pointer-events-none absolute left-0 z-20 truncate px-40 text-center text-[24px] font-semibold opacity-0 transition-opacity duration-300" style="right:130px;top:34px"></h1>
      <div id="al-body" class="absolute inset-0 text-[15px]" style="right:130px;padding-top:110px;color:var(--ink-soft)"><p class="px-[92px]">Chargement…</p></div>
    </section>`;
  const $ = (q: string) => el.querySelector<HTMLElement>("#" + q)!;
  $("al-back").onclick = () => (history.length > 1 ? history.back() : go("bibliotheque", "albums"));

  const play = async (trackID: string, shuffle: boolean) => {
    try {
      await api.PlayFrom(source, id, trackID, shuffle);
      go("platine");
    } catch (e) {
      const err = el.querySelector<HTMLElement>("#al-err");
      if (err) err.textContent = String(e);
    }
  };

  const render = (c: Collection) => {
    const tracks = c.tracks ?? [];
    const meta = [c.genre, c.year].filter(Boolean).join(" · ");
    const line = "color-mix(in srgb, var(--ink) 14%, transparent)";
    $("al-bar").textContent = c.title;
    $("al-body").innerHTML = `
      <div class="flex h-full gap-[6vw] pl-[92px] pr-10">
        <div class="vy-fly shrink-0 pt-6" style="width:min(36vw,560px)">
          <div class="aspect-square w-full bg-cover bg-center" style="background-color:#2a2220;${c.artworkUrl ? `background-image:${escapeHtml(cssUrl(c.artworkUrl))};` : ""}box-shadow:0 18px 40px rgba(20,10,6,.25);-webkit-box-reflect:below 2px linear-gradient(transparent 72%, rgba(0,0,0,.28))"></div>
        </div>
        <div id="al-scroll" class="shelf-scroll min-w-0 flex-1 overflow-y-auto pb-28" style="color:var(--ink)">
          <header id="al-head" class="vy-slide flex flex-col items-center gap-2 px-6 pb-12 pt-10 text-center">
            <h2 class="text-[44px] font-black leading-[1.08] tracking-tight">${escapeHtml(c.title)}</h2>
            <p class="mt-2 text-[28px] font-semibold opacity-90">${escapeHtml(c.artist)}</p>
            ${meta ? `<p class="mt-2 text-[20px] opacity-75">${escapeHtml(meta)}</p>` : ""}
            <div class="mt-9 flex w-full max-w-[640px] gap-6">
              <button id="al-play" class="flex-1 rounded-full border-2 py-4 text-[20px] font-semibold hover:bg-black/5" style="border-color:var(--ink)">Lecture</button>
              <button id="al-shuffle" class="flex-1 rounded-full border-2 py-4 text-[20px] font-semibold hover:bg-black/5" style="border-color:var(--ink)">Lecture aléatoire</button>
            </div>
            <p id="al-err" class="text-[14px] opacity-80"></p>
          </header>
          <ol class="px-2">${tracks.map((t, i) => `
            <li><button data-i="${i}" class="flex w-full items-center gap-5 border-b py-6 text-left text-[22px] hover:opacity-75" style="border-color:${line}">
              <span class="w-6 text-right text-[18px]">${t.loved ? "♥" : ""}</span>
              <span class="w-9 tabular-nums font-light opacity-80">${String(t.number || i + 1).padStart(2, "0")}</span>
              <span class="min-w-0 flex-1 truncate font-semibold">${escapeHtml(t.title)}</span>
              <span class="tabular-nums font-light opacity-80">${formatDuration(t.durationMs)}</span>
            </button></li>`).join("")}</ol>
          ${tracks.length === 0 ? `<p class="px-6 opacity-70">Aucun titre.</p>` : ""}
        </div>
      </div>`;
    $("al-play").onclick = () => play("", false);
    $("al-shuffle").onclick = () => play("", true);
    el.querySelectorAll<HTMLElement>("[data-i]").forEach((b) => (b.onclick = () => play(tracks[Number(b.dataset.i)]!.id, false)));
    // En défilant, l'en-tête laisse place à une barre de titre centrée (cf. playlistalbum.png).
    const sc = $("al-scroll");
    sc.addEventListener("scroll", () => $("al-bar").classList.toggle("opacity-0", sc.scrollTop < $("al-head").offsetHeight - 60), { passive: true });
    $("al-share").onclick = () => openShareCard({ title: c.title, subtitle: c.artist, artworkUrl: c.artworkUrl });
    // Fond et encre teintés par la pochette, limités à cette page.
    imageTint(c.artworkUrl).then((tint) => {
      if (!alive || !tint) return;
      const p = paletteFor(tint);
      const root = $("al-root");
      root.style.background = p.bg;
      root.style.setProperty("--ink", p.ink);
      root.style.setProperty("--ink-soft", p.inkSoft);
      root.style.setProperty("--glass", p.glass);
    });
  };

  const load = () => {
    api.Tracks(source, id).then((c) => { if (alive) render(c); }).catch((e) => {
      if (!alive) return;
      $("al-body").innerHTML = `<div class="px-[92px]" style="color:var(--ink)">${escapeHtml(String(e))} <button id="al-retry" class="underline">Réessayer</button></div>`;
      $("al-retry").onclick = load;
    });
  };
  load();
  return () => { alive = false; };
}
