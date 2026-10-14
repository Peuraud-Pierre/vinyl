import { activeSource, api, type ArtistPage, type Item } from "../api";
import { go } from "../router";
import { openShareCard } from "../share/sharecard";
import { escapeHtml, itemCard } from "./grid";

export function mount(el: HTMLElement, name: string): () => void {
  let alive = true;
  el.innerHTML = `<div class="p-10 pt-14 text-sm text-neutral-400">Chargement…</div>`;

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
      <header class="relative h-72 overflow-hidden">
        <div class="absolute inset-0 scale-110 bg-cover bg-center blur-2xl brightness-50" style="background-image:url('${escapeHtml(page.imageUrl)}')"></div>
        <div class="relative flex h-full items-end gap-6 p-10">
          <img src="${escapeHtml(page.imageUrl)}" class="h-40 w-40 rounded-full object-cover shadow-2xl" alt="">
          <div>
            <h1 class="text-4xl font-bold">${escapeHtml(page.name)}</h1>
            <div class="mt-4 flex gap-3">
              <button id="radio" class="rounded-full px-4 py-1.5 font-semibold text-neutral-900" style="background:var(--accent)">Radio de l'artiste</button>
              <button id="share" class="rounded-full border border-white/30 px-4 py-1.5">Partager</button>
            </div>
          </div>
        </div>
      </header>
      <section class="p-10 pt-6">
        <h2 class="mb-3 text-lg font-semibold">Meilleurs titres</h2>
        <ol id="tracks" class="flex flex-col">${tracks.map((t, i) => `
          <li><button data-id="${escapeHtml(t.id)}" class="flex w-full items-center gap-3 rounded px-2 py-1.5 text-left hover:bg-white/5">
            <span class="w-5 text-right text-neutral-500">${i + 1}</span>
            <img src="${escapeHtml(t.artworkUrl)}" class="h-10 w-10 rounded" alt="">
            <span class="flex-1 truncate">${escapeHtml(t.title)}</span>
            <span class="truncate text-xs text-neutral-500">${escapeHtml(t.subtitle)}</span>
          </button></li>`).join("")}</ol>
        <h2 class="mb-3 mt-8 text-lg font-semibold">Albums</h2>
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
    el.querySelector("#share")!.addEventListener("click", () => openShareCard({ title: page.name, subtitle: "", artworkUrl: page.imageUrl }));
  })();

  return () => {
    alive = false;
  };
}
