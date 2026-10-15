import { expect, test, vi } from "vitest";
import { mountDeck } from "./deck";

const deps = (shuffle = false) => ({
  previous: vi.fn(() => Promise.resolve()),
  next: vi.fn(() => Promise.resolve()),
  setShuffle: vi.fn((_on: boolean) => Promise.resolve()),
  modes: vi.fn(() => Promise.resolve({ shuffle, repeat: "off" })),
});
const flush = () => new Promise((r) => setTimeout(r, 0));

test("touches de platine : précédent et suivant pilotent le lecteur", async () => {
  const root = document.createElement("div");
  const d = deps();
  mountDeck(root, d);
  root.querySelector<HTMLElement>("[data-deck=prev]")!.click();
  root.querySelector<HTMLElement>("[data-deck=next]")!.click();
  expect(d.previous).toHaveBeenCalledTimes(1);
  expect(d.next).toHaveBeenCalledTimes(1);
});

test("aléatoire : le témoin suit l'état du lecteur et bascule tout de suite au clic", async () => {
  const root = document.createElement("div");
  const d = deps(true);
  const deck = mountDeck(root, d);
  await deck.refresh();
  const btn = root.querySelector<HTMLElement>("[data-deck=shuffle]")!;
  expect(btn.getAttribute("aria-pressed")).toBe("true");
  d.modes.mockImplementation(() => Promise.resolve({ shuffle: false, repeat: "off" }));
  btn.click();
  expect(btn.getAttribute("aria-pressed")).toBe("false"); // sans attendre le lecteur
  expect(d.setShuffle).toHaveBeenCalledWith(false);
  await flush(); await flush();
  expect(btn.getAttribute("aria-pressed")).toBe("false");
});

test("sans lecteur, les touches sont désactivées", () => {
  const root = document.createElement("div");
  const deck = mountDeck(root, deps());
  deck.setEnabled(false);
  root.querySelectorAll<HTMLButtonElement>("button").forEach((b) => expect(b.disabled).toBe(true));
  deck.setEnabled(true);
  root.querySelectorAll<HTMLButtonElement>("button").forEach((b) => expect(b.disabled).toBe(false));
});
