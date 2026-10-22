import { expect, test } from "vitest";
import { clampScroll, currentLine, plainProgress } from "./lyricsboard";

// Lignes inventées pour les tests.
const lines = [{ timeMs: 1000, text: "a" }, { timeMs: 4000, text: "b" }, { timeMs: 9000, text: "c" }];

test("ligne en cours : la dernière commencée", () => {
  expect(currentLine(lines, 0)).toBe(-1);
  expect(currentLine(lines, 1000)).toBe(0);
  expect(currentLine(lines, 5000)).toBe(1);
  expect(currentLine(lines, 60000)).toBe(2);
  expect(currentLine([], 5000)).toBe(-1);
});

test("texte simple : défile au fil du morceau, borné", () => {
  expect(plainProgress(0, 200000)).toBe(0);
  expect(plainProgress(200000, 200000)).toBe(1);
  expect(plainProgress(100000, 200000)).toBeCloseTo(0.5, 1);
  expect(plainProgress(5000, 0)).toBe(0);
});

test("défilement à la main : borné au texte", () => {
  expect(clampScroll(-40, 300)).toBe(0);
  expect(clampScroll(120, 300)).toBe(120);
  expect(clampScroll(900, 300)).toBe(300);
  expect(clampScroll(50, -10)).toBe(0);
});
