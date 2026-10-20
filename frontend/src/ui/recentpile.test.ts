import { expect, test, vi } from "vitest";
import type { HistoryEntry } from "../api";
import { mountRecentPile, PILE_TURN } from "./recentpile";

const e = (id: string): HistoryEntry => ({ source: "applemusic", trackId: id, title: "Titre " + id, artist: "Artiste", album: "", artworkUrl: "/artwork/" + id });

// Chaque disque est un groupe (.vy-recent-disc) de faces projetées ; son rang dans la pile est data-level.
const disc = (el: HTMLElement) => el.closest<HTMLElement>(".vy-recent-disc")!;
const bottom = (el: HTMLElement) => Number(disc(el).dataset.level);

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

test("chaque disque a sa pochette posée à plat (on la voit là où il dépasse de la pile)", () => {
  const root = document.createElement("div");
  mountRecentPile(root, { play: vi.fn() }).update(["3", "2", "1"].map(e));
  for (const id of ["1", "2", "3"]) {
    const lid = disc(root.querySelector<HTMLElement>(`[data-track="${id}"]`)!).querySelector<HTMLElement>(".vy-recent-lid")!;
    expect(lid.style.backgroundImage).toContain(`/artwork/${id}`);
  }
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
  const before = [at().dataset.level, at().dataset.yaw, at().dataset.dx];
  pile.update(["3", "2", "1"].map(e));
  expect([at().dataset.level, at().dataset.yaw, at().dataset.dx]).toEqual(before);
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
  expect(PILE_TURN).toBeLessThan(0);
  // Rotation de chaque disque = rotation de la pile + son propre écart (±9°).
  const yaws = Array.from(root.querySelectorAll<HTMLElement>(".vy-recent-disc")).map((d) => Number(d.dataset.yaw));
  expect(yaws.every((y) => Math.abs(y - PILE_TURN) <= 9)).toBe(true);
  expect(new Set(yaws).size).toBeGreaterThan(2); // pas tous pareils
});

// Dessinée à sa taille d'affichage (nette en grand ou zoomée) : pas d'agrandissement du conteneur ; les
// faces sont à l'échelle (--ps) et leurs matrices en tiennent compte. Redimensionner redessine.
test("pile dessinée à la taille d'affichage : pas d'agrandissement, faces à l'échelle", () => {
  const root = document.createElement("div");
  const pile = mountRecentPile(root, { play: vi.fn() });
  pile.setScale(0.8);
  pile.update(["b", "a"].map(e));
  const zoom = root.querySelector<HTMLElement>(".vy-recent-3d")!;
  expect(zoom.parentElement).toBe(root);
  expect(zoom.style.scale).toBe("");
  expect(zoom.style.getPropertyValue("--ps")).toBe("0.8");
  const lid = () => root.querySelector<HTMLElement>(".vy-recent-lid")!.style.transform;
  const before = lid();
  pile.setScale(1.1); // redimensionnement de la fenêtre
  expect(root.querySelector<HTMLElement>(".vy-recent-3d")!.style.getPropertyValue("--ps")).toBe("1.1");
  expect(lid()).not.toBe(before);
  expect(root.querySelectorAll(".vy-recent-disc").length).toBe(2);
});

test("côtés des disques assombris par la couleur, sans filtre (un filtre aplatit la 3D dans WebKit)", () => {
  const root = document.createElement("div");
  mountRecentPile(root, { play: vi.fn() }).update(["a"].map(e));
  const side = root.querySelector<HTMLElement>(".vy-recent-side")!;
  expect(side.style.background).toContain("color-mix");
  expect(side.style.filter).toBe("");
});

test("toute la pochette est cliquable : tranche, côté et pochette posée dessus (un seul lancement)", () => {
  const root = document.createElement("div");
  const play = vi.fn();
  mountRecentPile(root, { play }).update(["2", "1"].map(e));
  const d = disc(root.querySelector<HTMLElement>('[data-track="1"]')!);
  for (const part of [".vy-recent", ".vy-recent-side", ".vy-recent-lid"]) {
    play.mockClear();
    d.querySelector<HTMLElement>(part)!.click();
    expect(play).toHaveBeenCalledTimes(1);
    expect(play).toHaveBeenCalledWith(e("1"));
  }
});

test("réécouter un ancien disque ne vide pas la pile (comme Apple Music)", () => {
  const root = document.createElement("div");
  const play = vi.fn();
  const pile = mountRecentPile(root, { play });
  pile.update(["4", "3", "2", "1"].map(e));
  disc(root.querySelector<HTMLElement>('[data-track="2"]')!).querySelector<HTMLElement>(".vy-recent")!.click();
  expect(play).toHaveBeenCalledWith(e("2"));
  expect(root.querySelectorAll("[data-track]").length).toBe(4);
  root.querySelectorAll<HTMLElement>(".vy-recent-disc").forEach((d) => expect(d.dataset.leaving).toBeUndefined());
});

test("un nouveau disque posé sur la pile est animé (et lui seul)", () => {
  const root = document.createElement("div");
  const pile = mountRecentPile(root, { play: vi.fn() });
  pile.update(["1"].map(e));
  pile.update(["2", "1"].map(e));
  const at = (id: string) => disc(root.querySelector<HTMLElement>(`[data-track="${id}"]`)!);
  expect(at("2").dataset.dropped).toBe("1");
  expect(at("1").dataset.dropped).toBeUndefined();
});
