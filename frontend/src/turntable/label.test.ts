import { describe, expect, test } from "vitest";
import { labelColorFor, labelInk, labelMarkup } from "./label";

test("encre de l'étiquette : claire sur fond sombre, sombre sur fond clair", () => {
  expect(labelInk("#0b0b0b")).toBe("#f2ede8");
  expect(labelInk("#efe6c4")).toBe("#1d1714");
});

test("titre et artiste imprimés, échappés", () => {
  const html = labelMarkup({ title: "<b>Lumen</b>", artist: "Aurore & Vale", color: "#0b0b0b", ring: "VINYL · 33⅓ RPM" });
  expect(html).toContain("&lt;b&gt;Lumen&lt;/b&gt;");
  expect(html).toContain("Aurore &amp; Vale");
  expect(html).not.toContain("<b>Lumen");
});

test("titre très long tronqué avec points de suspension", () => {
  const html = labelMarkup({ title: "In Utero (20th Anniversary Edition) Remastered Deluxe", artist: "Nirvana", color: "#efe6c4", ring: "" });
  expect(html).toContain("…");
});

describe("couleur de l'étiquette", () => {
  test("texture noire : étiquette noire, quelle que soit la pochette", () => {
    expect(labelColorFor("black", "#c47a3a")).toBe("#0b0b0b");
  });
  test("picture disc : teinte de la pochette, crème par défaut", () => {
    expect(labelColorFor("picture", "#e9e3b0")).toBe("#e9e3b0");
    expect(labelColorFor("picture", null)).toBe("#efe6d2");
  });
  test("autres textures : noire", () => {
    for (const t of ["yolk", "split", "solid"]) expect(labelColorFor(t, "#123456")).toBe("#0b0b0b");
  });
});
