// Page album / playlist façon MD Vinyl (cf. screen/playlistalbum*.png) : pochette avec reflet,
// infos centrées, pilules « Lecture » / « Lecture aléatoire », liste des titres avec durées et cœurs.
import { api, type Collection } from "../api";
import { go, parseAlbumParam } from "../router";
import { openShareCard } from "../share/sharecard";
import { formatDuration } from "../ui/format";
import { icons } from "../ui/icons";
import { imageTint, paletteFor } from "../ui/theme";
import { escapeHtml } from "./grid";
import { matchingTracks, sortTracks, TRACK_ORDERS, type TrackOrder } from "./shelf";

const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;
const readOrder = (): TrackOrder => {
  try {
    const v = localStorage.getItem("vinyl.trackOrder") ?? "";
    return TRACK_ORDERS.some((o) => o.order === v) ? (v as TrackOrder) : "";
  } catch { return ""; }
};
const saveOrder = (o: TrackOrder) => { try { localStorage.setItem("vinyl.trackOrder", o); } catch { /* stockage indisponible */ } };

export function mount(el: HTMLElement, param: string): () => void {
  const { source, id } = parseAlbumParam(param);
  let alive = true;
  el.innerHTML = `
    <section id="al-root" class="relative h-screen overflow-hidden" style="color:var(--ink)">
      <button id="al-back" class="glass round-btn vy-drop fixed z-30" style="left:92px;top:28px" aria-label="Fermer">${icons.up}</button>
      <div class="vy-drop fixed z-30 flex items-center gap-3" style="right:51px;top:28px">
        <input id="al-q" type="search" class="field glass h-10 w-48" placeholder="Rechercher un titre" aria-label="Rechercher dans les titres" autocomplete="off" spellcheck="false">
        <button id="al-sort" class="glass round-btn" aria-label="Trier les titres" aria-expanded="false">${icons.sort}</button>
        <button id="al-share" class="glass round-btn" aria-label="Partager">${icons.share}</button>
      </div>
      <div id="al-sort-pop" class="glass fixed z-30 hidden w-56 rounded-[30px] p-2" style="right:113px;top:88px" role="menu" aria-label="Trier les titres">
        ${TRACK_ORDERS.map((o) => `<button data-order="${o.order}" class="menu-item" role="menuitemradio" aria-checked="false"><span class="w-4">✓</span>${o.label}</button>`).join("")}
      </div>
      <h1 id="al-bar" class="pointer-events-none absolute left-0 z-20 truncate px-40 text-center text-[24px] font-semibold opacity-0 transition-opacity duration-300" style="right:130px;top:34px"></h1>
      <div id="al-body" class="absolute inset-0 text-[15px]" style="right:130px;padding-top:110px;color:var(--ink-soft)"><p class="px-[92px]">Chargement…</p></div>
    </section>`;
  const $ = (q: string) => el.querySelector<HTMLElement>("#" + q)!;
  $("al-back").onclick = () => (history.length > 1 ? history.back() : go("bibliotheque", "albums"));
  // Tri des titres (année de sortie, durée) : retenu d'une page à l'autre ; appliqué dès que la liste est là.
  let order = readOrder();
  let applyOrder = () => {};
  const paintOrder = () => {
    $("al-sort").style.background = order ? "#15100e" : "";
    $("al-sort").style.color = order ? "#f6ece6" : "";
    el.querySelectorAll<HTMLElement>("[data-order]").forEach((b) => {
      const on = b.dataset.order === order;
      b.setAttribute("aria-checked", String(on));
      b.querySelector("span")!.style.visibility = on ? "visible" : "hidden";
    });
  };
  const toggleSort = (open = $("al-sort-pop").classList.contains("hidden")) => {
    $("al-sort-pop").classList.toggle("hidden", !open);
    $("al-sort").setAttribute("aria-expanded", String(open));
  };
  $("al-sort").onclick = (e) => { e.stopPropagation(); toggleSort(); };
  el.querySelectorAll<HTMLElement>("[data-order]").forEach((b) => (b.onclick = () => {
    order = b.dataset.order as TrackOrder;
    saveOrder(order);
    paintOrder();
    toggleSort(false);
    applyOrder();
  }));
  const closeSort = (e: MouseEvent) => { if (!(e.target instanceof Element && e.target.closest("#al-sort-pop"))) toggleSort(false); };
  document.addEventListener("click", closeSort);
  paintOrder();

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
    // Chanteur ou groupe sous chaque titre : toujours dans une playlist, dans un album seulement s'il diffère (featurings, compilations).
    const byline = (artist: string) => artist !== "" && (c.kind === "playlist" || artist !== c.artist);
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
          <ol id="al-list" class="px-2">${tracks.map((t, i) => `
            <li data-row="${i}"><button data-i="${i}" class="flex w-full items-center gap-5 border-b py-6 text-left text-[22px] hover:opacity-75" style="border-color:${line}">
              <span class="w-6 text-right text-[18px]">${t.loved ? "♥" : ""}</span>
              <span class="w-9 tabular-nums font-light opacity-80">${String(t.number || i + 1).padStart(2, "0")}</span>
              <span class="flex min-w-0 flex-1 flex-col">
                <span class="truncate font-semibold">${escapeHtml(t.title)}</span>
                ${byline(t.artist) ? `<span class="truncate text-[17px] font-normal opacity-75">${escapeHtml(t.artist)}</span>` : ""}
              </span>
              <span class="tabular-nums font-light opacity-80">${formatDuration(t.durationMs)}</span>
            </button></li>`).join("")}</ol>
          ${tracks.length === 0 ? `<p class="px-6 opacity-70">Aucun titre.</p>` : ""}
          <p id="al-none" class="hidden px-6 pt-6 opacity-70">Aucun titre ne correspond.</p>
        </div>
      </div>`;
    $("al-play").onclick = () => play("", false);
    $("al-shuffle").onclick = () => play("", true);
    el.querySelectorAll<HTMLElement>("[data-i]").forEach((b) => (b.onclick = () => play(tracks[Number(b.dataset.i)]!.id, false)));
    // En défilant, l'en-tête laisse place à une barre de titre centrée (cf. playlistalbum.png).
    const sc = $("al-scroll");
    sc.addEventListener("scroll", () => $("al-bar").classList.toggle("opacity-0", sc.scrollTop < $("al-head").offsetHeight - 60), { passive: true });
    // Recherche : n'affiche que les titres correspondants ; la liste remonte sous la barre de titre.
    const q = $("al-q") as HTMLInputElement;
    const rows = Array.from(el.querySelectorAll<HTMLElement>("[data-row]"));
    const filter = () => {
      const keep = new Set(matchingTracks(tracks, q.value));
      rows.forEach((r) => r.classList.toggle("hidden", !keep.has(Number(r.dataset.row))));
      $("al-none").classList.toggle("hidden", keep.size > 0 || tracks.length === 0);
      if (q.value.trim()) sc.scrollTop = Math.max(sc.scrollTop, $("al-head").offsetHeight - 60);
    };
    // Tri : les lignes sont replacées dans l'ordre choisi (numéros de piste inchangés).
    applyOrder = () => {
      const list = $("al-list");
      sortTracks(tracks, order).forEach((i) => list.appendChild(rows[i]!));
    };
    applyOrder();
    q.oninput = filter;
    q.onkeydown = (e) => { if (e.key === "Escape" && q.value) { q.value = ""; filter(); } };
    if (q.value) filter(); // texte tapé pendant le chargement
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
  return () => { alive = false; document.removeEventListener("click", closeSort); };
}
