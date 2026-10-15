import { expect, test, vi } from "vitest";
import type { HistoryEntry } from "../api";
import { mountRecentPile } from "./recentpile";

const e = (id: string): HistoryEntry => ({ source: "applemusic", trackId: id, title: "Titre " + id, artist: "Artiste", album: "", artworkUrl: "/artwork/" + id });

const bottom = (el: HTMLElement) => parseFloat(el.style.bottom);

test("pile sur la tranche : du bas vers le haut, le plus récent tout en haut", () => {
  const root = document.createElement("div");
  const pile = mountRecentPile(root, { play: vi.fn() });
  pile.update(["3", "2", "1"].map(e));
  const at = (id: string) => root.querySelector<HTMLElement>(`[data-track="${id}"]`)!;
  expect(bottom(at("1"))).toBe(0); // le plus ancien en bas
  expect(bottom(at("2"))).toBeGreaterThan(bottom(at("1")));
  expect(bottom(at("3"))).toBeGreaterThan(bottom(at("2")));
  expect(at("3").textContent).toContain("Titre 3"); // titre sur la tranche
});

test("la pochette du dessus est posée à plat sur la pile", () => {
  const root = document.createElement("div");
  mountRecentPile(root, { play: vi.fn() }).update(["3", "2"].map(e));
  const lid = root.querySelector<HTMLElement>(".vy-recent-lid")!;
  expect(lid.style.backgroundImage).toContain("/artwork/3");
});

test("au plus 7 pochettes", () => {
  const root = document.createElement("div");
  const pile = mountRecentPile(root, { play: vi.fn() });
  pile.update(["9", "8", "7", "6", "5", "4", "3", "2", "1"].map(e));
  expect(root.querySelectorAll("[data-track]").length).toBe(7);
});

test("un disque posé dessus ne fait pas bouger ceux d'en dessous", () => {
  const root = document.createElement("div");
  const pile = mountRecentPile(root, { play: vi.fn() });
  pile.update(["2", "1"].map(e));
  const at = () => root.querySelector<HTMLElement>('[data-track="2"]')!;
  const before = [at().style.transform, at().style.bottom];
  pile.update(["3", "2", "1"].map(e));
  expect([at().style.transform, at().style.bottom]).toEqual(before);
});

test("un clic sur une pochette la relance", () => {
  const root = document.createElement("div");
  const play = vi.fn();
  mountRecentPile(root, { play }).update(["2", "1"].map(e));
  root.querySelector<HTMLElement>('[data-track="1"]')!.click();
  expect(play).toHaveBeenCalledWith(e("1"));
});

test("pile vide : rien d'affiché", () => {
  const root = document.createElement("div");
  mountRecentPile(root, { play: vi.fn() }).update([]);
  expect(root.querySelectorAll("[data-track]").length).toBe(0);
  expect(root.querySelector(".vy-recent-lid")).toBeNull();
});
