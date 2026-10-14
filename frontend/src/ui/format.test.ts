import { expect, test } from "vitest";
import { formatDuration } from "./format";

test("minutes et secondes façon MD Vinyl", () => {
  expect(formatDuration(217000)).toBe("3′37″");
  expect(formatDuration(59000)).toBe("0′59″");
  expect(formatDuration(0)).toBe("0′00″");
});
test("plus d'une heure", () => {
  expect(formatDuration(3_725_000)).toBe("1h02′05″");
});
test("valeurs invalides", () => {
  expect(formatDuration(NaN)).toBe("");
  expect(formatDuration(-5)).toBe("");
});
