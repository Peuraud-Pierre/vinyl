import { expect, test } from "vitest";
import { labelInk, labelMarkup } from "./label";

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
