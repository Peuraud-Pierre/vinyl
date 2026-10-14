import { expect, test } from "vitest";
import { filterItems, spineColors } from "./shelf";

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
