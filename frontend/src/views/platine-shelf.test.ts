import { expect, test, vi } from "vitest";
import type { Item } from "../api";
import { wireShelf } from "./platine";

const pl = (i: number): Item => ({ id: `playlist:${i}`, kind: "playlist", title: `P${i}`, subtitle: "", artworkUrl: "", url: "" });
const flush = () => new Promise((r) => setTimeout(r, 0));

test("l'étagère charge les playlists de la source active et se recharge au changement de source", async () => {
  let source = "applemusic";
  let notify!: (s: { source: string }) => void;
  const library = vi.fn((s: string) => Promise.resolve(s === "applemusic" ? [pl(1), pl(2)] : [pl(7), pl(8), pl(9)]));
  const el = document.createElement("div");
  wireShelf(el, {
    source: () => Promise.resolve(source), library, current: () => Promise.resolve(""),
    onSettings: (cb) => { notify = cb; return () => {}; },
  });
  await flush();
  expect(library).toHaveBeenLastCalledWith("applemusic");
  expect(el.querySelectorAll(".vy-ws-spine").length).toBe(1);
  source = "spotify";
  notify({ source: "spotify" });
  await flush();
  expect(library).toHaveBeenLastCalledWith("spotify");
  expect(el.querySelectorAll(".vy-ws-spine").length).toBe(2);
  // Un autre réglage (même source) ne recharge pas.
  notify({ source: "spotify" });
  await flush();
  expect(library).toHaveBeenCalledTimes(2);
});

test("clic : vue Étagère des playlists de la source active", async () => {
  const nav = vi.fn();
  const el = document.createElement("div");
  wireShelf(el, { source: () => Promise.resolve("spotify"), library: () => Promise.resolve([]), current: () => Promise.resolve(""), nav, onSettings: () => () => {} });
  await flush();
  el.click();
  await flush();
  expect(nav).toHaveBeenCalledWith("bibliotheque", "playlists");
});
