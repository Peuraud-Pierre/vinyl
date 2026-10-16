import { expect, test, vi } from "vitest";
import type { Item } from "../api";
import { mountWallShelf, openPlaylistShelf, SHELF_MAX } from "./wallshelf";

const pl = (i: number, title = `Playlist ${i}`): Item => ({ id: `playlist:${i}`, kind: "playlist", title, subtitle: "", artworkUrl: `/art/${i}`, url: "" });
const flush = () => new Promise((r) => setTimeout(r, 0));
const deps = (items: Item[], current = "") => ({ load: vi.fn(() => Promise.resolve(items)), current: vi.fn(() => Promise.resolve(current)), open: vi.fn() });

test("12 playlists au plus : 11 tranches + 1 pochette en biais (la première par défaut)", async () => {
  const el = document.createElement("div");
  mountWallShelf(el, deps(Array.from({ length: 15 }, (_, i) => pl(i))));
  await flush();
  expect(SHELF_MAX).toBe(12);
  expect(el.querySelectorAll(".vy-ws-spine").length).toBe(11);
  expect(el.querySelector<HTMLElement>(".vy-ws-cover")!.style.backgroundImage).toContain("/art/0");
});

test("pochette en biais = playlist en cours si elle est sur l'étagère, jamais répétée en tranche", async () => {
  const el = document.createElement("div");
  mountWallShelf(el, deps(Array.from({ length: 5 }, (_, i) => pl(i)), "playlist:3"));
  await flush();
  expect(el.querySelector<HTMLElement>(".vy-ws-cover")!.style.backgroundImage).toContain("/art/3");
  const titles = Array.from(el.querySelectorAll<HTMLElement>(".vy-ws-spine"), (s) => s.title);
  expect(titles).not.toContain("Playlist 3");
  expect(titles.length).toBe(4);
});

test("playlist en cours absente de l'étagère : la première", async () => {
  const el = document.createElement("div");
  mountWallShelf(el, deps([pl(0), pl(1)], "playlist:99"));
  await flush();
  expect(el.querySelector<HTMLElement>(".vy-ws-cover")!.style.backgroundImage).toContain("/art/0");
});

test("clic n'importe où ou clavier : open, une fois par geste, même après refresh", async () => {
  const el = document.createElement("div");
  const d = deps([pl(0), pl(1)]);
  const shelf = mountWallShelf(el, d);
  await flush();
  shelf.refresh();
  await flush();
  el.querySelector<HTMLElement>(".vy-ws-spine")!.click();
  expect(d.open).toHaveBeenCalledTimes(1);
  el.querySelector<HTMLElement>(".vy-ws-board")!.click();
  el.dispatchEvent(new KeyboardEvent("keydown", { key: "Enter" }));
  el.dispatchEvent(new KeyboardEvent("keydown", { key: " " }));
  expect(d.open).toHaveBeenCalledTimes(4);
  expect(el.getAttribute("role")).toBe("button");
  expect(el.getAttribute("aria-label")).toBe("Ouvrir l'étagère des playlists");
  expect(el.title).toBe("Ouvrir l'étagère");
});

test("liste vide ou erreur : planche vide, toujours cliquable, sans message", async () => {
  for (const load of [() => Promise.resolve([] as Item[]), () => Promise.reject(new Error("403"))]) {
    const el = document.createElement("div");
    const open = vi.fn();
    mountWallShelf(el, { load, current: () => Promise.resolve(""), open });
    await flush();
    expect(el.querySelector(".vy-ws-board")).not.toBeNull();
    expect(el.querySelectorAll(".vy-ws-spine, .vy-ws-cover").length).toBe(0);
    expect(el.textContent?.trim()).toBe("");
    el.click();
    expect(open).toHaveBeenCalledTimes(1);
  }
});

test("titre piégé : pas d'injection HTML", async () => {
  const el = document.createElement("div");
  mountWallShelf(el, deps([pl(0), pl(1, "<b>&</b>")]));
  await flush();
  const spine = el.querySelector<HTMLElement>(".vy-ws-spine")!;
  expect(spine.querySelector("b")).toBeNull();
  expect(spine.title).toBe("<b>&</b>");
});

test("démontée avant la fin du chargement : rien n'est écrit", async () => {
  const el = document.createElement("div");
  let resolve!: (l: Item[]) => void;
  const shelf = mountWallShelf(el, { load: () => new Promise((r) => (resolve = r)), current: () => Promise.resolve(""), open: vi.fn() });
  shelf.destroy();
  resolve([pl(0), pl(1)]);
  await flush();
  expect(el.querySelectorAll(".vy-ws-spine").length).toBe(0);
});

test("openPlaylistShelf : vue étagère forcée, position oubliée, playlists", () => {
  localStorage.setItem("vinyl.shelf", "flat");
  const nav = vi.fn();
  openPlaylistShelf("spotify", nav);
  expect(localStorage.getItem("vinyl.shelf")).toBe("side");
  expect(nav).toHaveBeenCalledWith("bibliotheque", "playlists");
});
