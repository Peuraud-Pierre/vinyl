import { expect, test } from "vitest";
import { filterItems, PILE_SIZE, PILE_STEP, pileCurrent, pilePad, settleWithin, spineColors } from "./shelf";

const items = [
  { id: "1", kind: "album", title: "Lumen", subtitle: "Aurore Vale", artworkUrl: "", url: "" },
  { id: "2", kind: "album", title: "Pavés", subtitle: "Gambi Sud", artworkUrl: "", url: "" },
  { id: "3", kind: "album", title: "Écho", subtitle: "Sable", artworkUrl: "", url: "" },
];

test("recherche : titre ou artiste, sans tenir compte des accents ni de la casse", () => {
  expect(filterItems(items, "").map((i) => i.id)).toEqual(["1", "2", "3"]);
  expect(filterItems(items, "aurore").map((i) => i.id)).toEqual(["1"]);
  expect(filterItems(items, "PAVES").map((i) => i.id)).toEqual(["2"]);
  expect(filterItems(items, "echo").map((i) => i.id)).toEqual(["3"]);
  expect(filterItems(items, "zzz")).toEqual([]);
});

test("couleur de tranche stable pour un même titre, encre lisible", () => {
  const a = spineColors("Lumen");
  expect(spineColors("Lumen")).toEqual(a);
  expect(a.bg).toMatch(/^#[0-9a-f]{6}$/);
  expect(["#1b1513", "#f6efe9"]).toContain(a.ink);
});

test("couleur de tranche depuis la couleur de la pochette", () => {
  expect(spineColors("x", "#efe6c4").ink).toBe("#1b1513");
  expect(spineColors("x", "#101010").ink).toBe("#f6efe9");
});

test("boîtier prêt (pochette + teinte) : on attend, mais jamais plus que le délai", async () => {
  expect(await settleWithin(Promise.resolve("prêt"), 1000)).toBe(true);
  expect(await settleWithin(Promise.reject(new Error("x")), 1000)).toBe(true); // échec : on affiche quand même
  const t0 = Date.now();
  expect(await settleWithin(new Promise(() => {}), 50)).toBe(false); // jamais prêt : affiché après le délai
  expect(Date.now() - t0).toBeLessThan(500);
});

// Vue pile (demande utilisateur) : la pochette au milieu de l'écran sort à droite ; le premier et le
// dernier élément peuvent venir au milieu.
test("vue pile : marge pour amener le premier et le dernier au milieu", () => {
  expect(pilePad(800)).toBe(400 - PILE_SIZE / 2);
  expect(pilePad(200)).toBe(0); // fenêtre minuscule : pas de marge négative
});
test("vue pile : l'élément courant est celui dont la pochette est au milieu de l'écran", () => {
  const h = 800, pad = pilePad(h);
  expect(pileCurrent(0, h, pad, 50)).toBe(0); // au départ : le premier
  expect(pileCurrent(PILE_STEP * 3, h, pad, 50)).toBe(3);
  expect(pileCurrent(PILE_STEP * 3 + PILE_STEP * 0.4, h, pad, 50)).toBe(3); // arrondi au plus proche
  expect(pileCurrent(PILE_STEP * 3 + PILE_STEP * 0.6, h, pad, 50)).toBe(4);
  expect(pileCurrent(1e6, h, pad, 50)).toBe(49); // jamais au-delà du dernier
  expect(pileCurrent(0, h, pad, 0)).toBe(-1); // liste vide
});
test("vue pile : tout en bas, le dernier élément est au milieu", () => {
  const h = 800, pad = pilePad(h), n = 30;
  // Hauteur totale : marges + pile (chaque élément ne dépasse que de PILE_STEP, sauf le dernier, entier).
  const scrollHeight = pad + (n - 1) * PILE_STEP + PILE_SIZE + pad;
  expect(pileCurrent(scrollHeight - h, h, pad, n)).toBe(n - 1);
});
