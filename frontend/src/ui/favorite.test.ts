import { expect, test } from "vitest";
import type { NowPlaying } from "../api";
import { mountFavorite } from "./favorite";

const np = (trackId: string) => ({ source: "spotify", trackId, title: "Titre", artist: "Artiste" }) as NowPlaying;
const flush = () => new Promise((r) => setTimeout(r, 0));

test("relu à chaque morceau, basculé au clic", async () => {
  const btn = document.createElement("button");
  const sets: boolean[] = [];
  const fav = mountFavorite(btn, { get: async (n) => n.trackId === "1", set: async (_, on) => { sets.push(on); } });
  fav.update(np("1"));
  await flush();
  expect(btn.getAttribute("aria-pressed")).toBe("true");
  btn.click();
  expect(btn.getAttribute("aria-pressed")).toBe("false");
  expect(sets).toEqual([false]);
  fav.update(np("2"));
  await flush();
  expect(btn.getAttribute("aria-pressed")).toBe("false");
});

test("échec : retour en arrière avec l'explication", async () => {
  const btn = document.createElement("button");
  const fav = mountFavorite(btn, { get: async () => false, set: async () => { throw new Error("reconnecte Spotify"); } });
  fav.update(np("1"));
  await flush();
  btn.click();
  expect(btn.getAttribute("aria-pressed")).toBe("true");
  await flush();
  expect(btn.getAttribute("aria-pressed")).toBe("false");
  expect(btn.title).toBe("reconnecte Spotify");
});

test("rien en lecture : caché", () => {
  const btn = document.createElement("button");
  mountFavorite(btn, { get: async () => false, set: async () => {} }).update(null);
  expect(btn.hidden).toBe(true);
});
