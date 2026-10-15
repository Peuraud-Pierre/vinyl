import { expect, test, vi } from "vitest";
import type { HistoryEntry } from "../api";
import { mountRecentPile, PILE_TURN } from "./recentpile";

const e = (id: string): HistoryEntry => ({ source: "applemusic", trackId: id, title: "Titre " + id, artist: "Artiste", album: "", artworkUrl: "/artwork/" + id });

// Chaque disque est un petit volume (.vy-recent-disc) : sa hauteur dans la pile est portée par lui.
const disc = (el: HTMLElement) => el.closest<HTMLElement>(".vy-recent-disc")!;
const bottom = (el: HTMLElement) => parseFloat(disc(el).style.bottom);

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
  const at = () => disc(root.querySelector<HTMLElement>('[data-track="2"]')!);
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

test("pile tournée dans l'autre sens, chaque disque un peu de travers au hasard (mais toujours pareil)", () => {
  const root = document.createElement("div");
  mountRecentPile(root, { play: vi.fn() }).update(["a", "b", "c", "d", "e", "f", "g"].map(e));
  expect(root.querySelector<HTMLElement>(".vy-recent-3d")!.style.transform).toBe(`rotateY(${PILE_TURN}deg)`);
  expect(PILE_TURN).toBeLessThan(0);
  const yaws = Array.from(root.querySelectorAll<HTMLElement>(".vy-recent-disc")).map((d) => Number(/rotateY\((-?[\d.]+)deg\)/.exec(d.style.transform)![1]));
  expect(yaws.every((y) => Math.abs(y) <= 10)).toBe(true);
  expect(new Set(yaws).size).toBeGreaterThan(2); // pas tous pareils
});
