import { expect, test, vi } from "vitest";
import type { Item } from "../api";
import { wireShelf } from "./platine";

const pl = (i: number, art = ""): Item => ({ id: `playlist:${i}`, kind: "playlist", title: `P${i}`, subtitle: "", artworkUrl: art, url: "" });
const flush = () => new Promise((r) => setTimeout(r, 0));

// Faux branchements : source résolue (activeSource), réglages et instantanés du lecteur.
function harness(resolved: () => string, library: (s: string) => Promise<Item[]>, extra: Partial<Parameters<typeof wireShelf>[1]> = {}) {
  let settings!: (s: { source: string }) => void;
  let snapshot!: (s: { nowPlaying: { source: string } }) => void;
  const el = document.createElement("div");
  const w = wireShelf(el, {
    source: () => Promise.resolve(resolved()), library, current: () => Promise.resolve(""),
    onSettings: (cb) => { settings = cb; return () => {}; },
    onSnapshot: (cb) => { snapshot = cb; return () => {}; },
    cache: new Map(),
    ...extra,
  });
  return { el, w, settings: (s: string) => settings({ source: s }), snapshot: (s: string) => snapshot({ nowPlaying: { source: s } }) };
}

test("l'étagère charge les playlists de la source active et se recharge au changement de source", async () => {
  let source = "applemusic";
  const library = vi.fn((s: string) => Promise.resolve(s === "applemusic" ? [pl(1, "/a"), pl(2, "/a")] : [pl(7, "/a"), pl(8, "/a"), pl(9, "/a")]));
  const h = harness(() => source, library);
  await flush();
  expect(library).toHaveBeenLastCalledWith("applemusic");
  expect(h.el.querySelectorAll(".vy-ws-spine").length).toBe(1);
  source = "spotify";
  h.settings("spotify");
  await flush();
  expect(library).toHaveBeenLastCalledWith("spotify");
  expect(h.el.querySelectorAll(".vy-ws-spine").length).toBe(2);
  // Un autre réglage (même source) ne recharge pas.
  h.settings("spotify");
  await flush();
  expect(library).toHaveBeenCalledTimes(2);
});

// Réglage « auto » (par défaut) : seule la source réellement utilisée compte.
test("mode auto : un réglage sans changement de source ne recharge pas ; un changement de lecteur, si", async () => {
  let source = "applemusic";
  const library = vi.fn((s: string) => Promise.resolve([pl(s === "applemusic" ? 1 : 7, "/a"), pl(2, "/a")]));
  const h = harness(() => source, library);
  await flush();
  h.settings("auto"); // craquement modifié, par exemple
  await flush();
  expect(library).toHaveBeenCalledTimes(1);
  source = "spotify"; // Musique arrêté, Spotify lancé : aucun événement de réglage
  h.snapshot("spotify");
  await flush();
  expect(library).toHaveBeenLastCalledWith("spotify");
  h.snapshot("spotify"); // instantanés suivants : rien de plus
  await flush();
  expect(library).toHaveBeenCalledTimes(2);
});

// Apple Music ne donne pas les pochettes avec la liste : elles sont demandées pour les playlists affichées.
test("pochettes complétées pour les playlists affichées", async () => {
  const artwork = vi.fn((_s: string, id: string) => Promise.resolve(`/art/${id}`));
  const h = harness(() => "applemusic", () => Promise.resolve(Array.from({ length: 15 }, (_, i) => pl(i))), { artwork });
  await flush();
  await flush();
  expect(artwork).toHaveBeenCalledTimes(12);
  expect(h.el.querySelector<HTMLElement>(".vy-ws-cover")!.style.backgroundImage).toContain("/art/playlist:0");
});

// Liste gardée en mémoire tant que l'app tourne : un nouvel affichage de la platine ne relit pas la bibliothèque.
test("liste gardée en mémoire entre deux affichages", async () => {
  const cache = new Map<string, Item[]>();
  const library = vi.fn(() => Promise.resolve([pl(1, "/a"), pl(2, "/a")]));
  harness(() => "applemusic", library, { cache }).w.destroy();
  await flush();
  const again = harness(() => "applemusic", library, { cache });
  await flush();
  expect(library).toHaveBeenCalledTimes(1);
  expect(again.el.querySelectorAll(".vy-ws-spine").length).toBe(1);
});

test("clic : vue Étagère des playlists de la source active", async () => {
  const nav = vi.fn();
  const h = harness(() => "spotify", () => Promise.resolve([]), { nav });
  await flush();
  h.el.click();
  await flush();
  expect(nav).toHaveBeenCalledWith("bibliotheque", "playlists");
});
