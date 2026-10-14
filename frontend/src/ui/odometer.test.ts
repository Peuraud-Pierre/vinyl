import { expect, test } from "vitest";
import { odometerDigits } from "./odometer";

test("8 chiffres avec zéros à gauche", () => {
  expect(odometerDigits(34)).toEqual([0, 0, 0, 0, 0, 0, 3, 4]);
});
test("bornes : négatif → 0, trop grand → 99999999", () => {
  expect(odometerDigits(-5)).toEqual([0, 0, 0, 0, 0, 0, 0, 0]);
  expect(odometerDigits(1234567890)).toEqual([9, 9, 9, 9, 9, 9, 9, 9]);
});
test("non entier ou NaN", () => {
  expect(odometerDigits(12.9)).toEqual([0, 0, 0, 0, 0, 0, 1, 2]);
  expect(odometerDigits(NaN)).toEqual([0, 0, 0, 0, 0, 0, 0, 0]);
});
test("largeur personnalisée", () => {
  expect(odometerDigits(7, 3)).toEqual([0, 0, 7]);
});
