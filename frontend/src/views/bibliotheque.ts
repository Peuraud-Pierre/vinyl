import { activeSource, api, type Item } from "../api";
import { go } from "../router";
import { escapeHtml, itemCard } from "./grid";

const TITLES: Record<string, string> = { playlists: "Playlists", albums: "Albums", likes: "Titres likés", artistes: "Artistes" };
const KIND: Record<string, string> = { playlists: "playlists", albums: "albums", likes: "likes", artistes: "artists" };

export function mount(el: HTMLElement, route: string): () => void {
  let alive = true;
  el.innerHTML = `<div class="p-8 pt-14">
    <h1 class="mb-6 text-2xl font-semibold">${TITLES[route] ?? ""}</h1>
    <div id="grid" class="grid grid-cols-[repeat(auto-fill,minmax(150px,1fr))] gap-5"></div>
    <div id="status" class="mt-6 text-sm text-neutral-400"></div>
    <div id="sentinel" class="h-4"></div></div>`;
  const grid = el.querySelector<HTMLElement>("#grid")!;
  const status = el.querySelector<HTMLElement>("#status")!;
  const items = new Map<string, Item>();
  let source: "applemusic" | "spotify" = "applemusic";
  let next = 0;
  let loading = false;

  const showError = (e: unknown) => {
    const msg = String(e);
    if (msg.includes("non connecté") || msg.includes("reconnecte")) {
      status.innerHTML = `Connecte ton compte Spotify pour voir ta bibliothèque. <button id="go-settings" class="underline">Ouvrir les Réglages</button>`;
      status.querySelector("#go-settings")!.addEventListener("click", () => go("reglages"));
    } else if (msg.includes("non disponible")) {
      status.textContent = "Non disponible pour Apple Music.";
    } else if (msg.includes("non lancé")) {
      status.textContent = "Ouvre l'app Musique pour voir ta bibliothèque.";
    } else if (msg.includes("non autorisée")) {
      status.innerHTML = `Vinyl n'a pas l'autorisation de piloter Musique. <button id="open-auto" class="underline">Ouvrir les réglages</button>`;
      status.querySelector("#open-auto")!.addEventListener("click", () => api.OpenAutomationSettings());
    } else {
      status.innerHTML = `${escapeHtml(msg)} <button id="retry" class="underline">Réessayer</button>`;
      status.querySelector("#retry")!.addEventListener("click", load);
    }
  };

  async function load() {
    if (loading || next < 0 || !alive) return;
    loading = true;
    status.textContent = "Chargement…";
    try {
      const page = await api.Library(source, KIND[route] ?? "", next);
      if (!alive) return;
      for (const it of page.items ?? []) {
        items.set(it.id, it);
        grid.insertAdjacentHTML("beforeend", itemCard(it));
        if (!it.artworkUrl && source === "applemusic" && it.kind !== "artist") lazyArtwork(it);
      }
      next = page.next;
      status.textContent = items.size === 0 ? "Rien ici pour l'instant." : "";
    } catch (e) {
      showError(e);
    } finally {
      loading = false;
    }
  }

  async function lazyArtwork(it: Item) {
    const url = await api.ItemArtwork(source, it.id).catch(() => "");
    const card = grid.querySelector<HTMLElement>(`[data-id="${CSS.escape(it.id)}"] [data-art]`);
    if (url && card) card.outerHTML = `<img src="${escapeHtml(url)}" class="aspect-square w-full rounded-md object-cover" alt="">`;
  }

  grid.addEventListener("click", async (e) => {
    const btn = (e.target as HTMLElement).closest<HTMLElement>("[data-id]");
    const it = btn && items.get(btn.dataset.id!);
    if (!it) return;
    if (it.kind === "artist") return go("artiste", it.title);
    try {
      await api.PlayItem(source, it.id);
      go("platine");
    } catch (err) {
      showError(err);
    }
  });

  const observer = new IntersectionObserver((entries) => {
    if (entries.some((x) => x.isIntersecting)) load();
  });
  activeSource().then((s) => {
    source = s;
    if (route === "likes" && s !== "spotify") {
      status.textContent = "Les titres likés sont disponibles avec Spotify.";
      return;
    }
    observer.observe(el.querySelector("#sentinel")!);
    load();
  });
  return () => {
    alive = false;
    observer.disconnect();
  };
}
