import { expect, test } from "vitest";
import { estimatePosition } from "./clock";

test("avance pendant la lecture", () => {
  expect(estimatePosition(10_000, 200_000, true, 1000, 1500)).toBe(10_500);
});
test("figée en pause", () => {
  expect(estimatePosition(10_000, 200_000, false, 1000, 9000)).toBe(10_000);
});
test("bornée à la durée", () => {
  expect(estimatePosition(199_000, 200_000, true, 0, 5000)).toBe(200_000);
});
test("durée inconnue : avance sans borne", () => {
  expect(estimatePosition(1000, 0, true, 0, 1000)).toBe(2000);
});
