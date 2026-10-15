import { expect, test, vi } from "vitest";
import type { HistoryEntry } from "../api";
import { mountRecentPile } from "./recentpile";

const e = (id: string): HistoryEntry => ({ source: "applemusic", trackId: id, title: "Titre " + id, artist: "Artiste", album: "", artworkUrl: "/artwork/" + id });

test("pile des derniers écoutés : le plus récent est posé au-dessus (dernier dans le DOM)", () => {
  const root = document.createElement("div");
  const pile = mountRecentPile(root, { play: vi.fn() });
  pile.update(["3", "2", "1"].map(e));
  const sleeves = root.querySelectorAll<HTMLElement>("[data-track]");
  expect(sleeves.length).toBe(3);
  expect(sleeves[sleeves.length - 1]!.dataset.track).toBe("3");
  expect(sleeves[sleeves.length - 1]!.getAttribute("aria-label")).toContain("Titre 3");
});

test("au plus 7 pochettes", () => {
  const root = document.createElement("div");
  const pile = mountRecentPile(root, { play: vi.fn() });
  pile.update(["9", "8", "7", "6", "5", "4", "3", "2", "1"].map(e));
  expect(root.querySelectorAll("[data-track]").length).toBe(7);
});

test("chaque pochette garde la même position d'une mise à jour à l'autre (pile naturelle, sans tressauter)", () => {
  const root = document.createElement("div");
  const pile = mountRecentPile(root, { play: vi.fn() });
  pile.update(["2", "1"].map(e));
  const before = root.querySelector<HTMLElement>('[data-track="2"]')!.style.transform;
  pile.update(["3", "2", "1"].map(e));
  expect(root.querySelector<HTMLElement>('[data-track="2"]')!.style.transform).toBe(before);
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
});
