import { activeSource, api, type ArtistPage, type Item } from "../api";
import { go } from "../router";
import { openShareCard } from "../share/sharecard";
import { escapeHtml, itemCard } from "./grid";
import { icons } from "../ui/icons";

export function mount(el: HTMLElement, name: string): () => void {
  let alive = true;
  el.innerHTML = `<div class="p-10 pt-24 text-[15px]" style="color:var(--ink-soft)">Chargement…</div>`;

  (async () => {
    const source = await activeSource();
    const snap = await api.GetSnapshot();
    const trackId = snap.nowPlaying.artist === name ? snap.nowPlaying.trackId : "";
    let page: ArtistPage;
    try {
      page = await api.Artist(source, name, trackId);
    } catch (e) {
      if (!alive) return;
      el.innerHTML = `<div class="p-10 pt-14 text-sm text-neutral-300">${escapeHtml(String(e))}
        <button id="retry" class="ml-2 underline">Réessayer</button></div>`;
      el.querySelector("#retry")!.addEventListener("click", () => mount(el, name));
      return;
    }
    if (!alive) return;
    const tracks = page.topTracks ?? [];
    const albums = page.albums ?? [];
    const all = new Map<string, Item>([...tracks, ...albums].map((i) => [i.id, i]));
    el.innerHTML = `
      <button id="back" class="glass round-btn fixed z-30" style="left:96px;top:30px" aria-label="Retour">${icons.back}</button>
      <header class="flex items-end gap-8 px-[96px] pb-6 pt-28" style="padding-right:190px">
        <img src="${escapeHtml(page.imageUrl)}" class="h-48 w-48 rounded-full object-cover" style="box-shadow:0 20px 40px rgba(20,8,6,.35)" alt="">
        <div>
          <h1 class="text-[44px] font-extrabold leading-tight">${escapeHtml(page.name)}</h1>
          <div class="mt-5 flex gap-4">
            <button id="radio" class="pill h-12 cursor-pointer border-2 px-6 text-[17px] font-bold" style="border-color:var(--ink)">Radio de l'artiste</button>
            <button id="share" class="pill h-12 cursor-pointer border-2 px-6 text-[17px] font-bold" style="border-color:var(--ink)">Partager</button>
          </div>
        </div>
      </header>
      <section class="px-[96px] pb-16" style="padding-right:190px">
        <h2 class="mb-3 text-[22px] font-bold">Meilleurs titres</h2>
        <ol id="tracks" class="flex flex-col">${tracks.map((t, i) => `
          <li><button data-id="${escapeHtml(t.id)}" class="flex w-full cursor-pointer items-center gap-4 border-b px-2 py-3 text-left text-[17px] hover:bg-white/10" style="border-color:color-mix(in srgb, var(--ink) 14%, transparent)">
            <span class="w-7 tabular-nums" style="color:var(--ink-soft)">${String(i + 1).padStart(2, "0")}</span>
            <img src="${escapeHtml(t.artworkUrl)}" class="h-10 w-10 rounded" alt="">
            <span class="flex-1 truncate font-semibold">${escapeHtml(t.title)}</span>
            <span class="truncate text-[14px]" style="color:var(--ink-soft)">${escapeHtml(t.subtitle)}</span>
          </button></li>`).join("")}</ol>
        <h2 class="mb-4 mt-10 text-[22px] font-bold">Albums</h2>
        <div id="albums" class="grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-5">${albums.map(itemCard).join("")}</div>
      </section>`;

    const pick = (e: Event) => all.get((e.target as HTMLElement).closest<HTMLElement>("[data-id]")?.dataset.id ?? "");
    el.querySelector("#tracks")!.addEventListener("click", async (e) => {
      const it = pick(e);
      if (!it) return;
      await api.PlayCatalogTrack(source, it, page.name);
      go("platine");
    });
    el.querySelector("#albums")!.addEventListener("click", async (e) => {
      const it = pick(e);
      if (!it) return;
      if (source === "spotify") {
        await api.PlayItem("spotify", it.id);
        go("platine");
      } else await api.OpenURL(it.url);
    });
    el.querySelector("#radio")!.addEventListener("click", async () => {
      if (source === "spotify") {
        // Pas de radio fiable dans l'API : on lance les meilleurs titres à partir du premier.
        if (tracks[0]) {
          await api.PlayItem("spotify", tracks[0].id);
          go("platine");
        }
      } else await api.OpenURL(page.url); // page artiste dans Musique, qui propose sa station
    });
    el.querySelector("#back")!.addEventListener("click", () => history.back());
    el.querySelector("#share")!.addEventListener("click", () => openShareCard({ title: page.name, subtitle: "", artworkUrl: page.imageUrl }));
  })();

  return () => {
    alive = false;
  };
}
