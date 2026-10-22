import { expect, test } from "vitest";
import { MS_PER_DEGREE, pointerAngle, scratchTarget, turnDelta } from "./scratch";

test("angle du pointeur autour du disque : 0 en haut, 90 à droite", () => {
  expect(pointerAngle(0, 0, 0, -10)).toBeCloseTo(0);
  expect(pointerAngle(0, 0, 10, 0)).toBeCloseTo(90);
  expect(pointerAngle(0, 0, -10, 0)).toBeCloseTo(-90);
});

test("écart d'angle le plus court, sans saut en passant le bas", () => {
  expect(turnDelta(170, -170)).toBe(20);
  expect(turnDelta(-170, 170)).toBe(-20);
  expect(turnDelta(10, 40)).toBe(30);
});

test("un tour = 1,8 s ; borné au début et avant la fin du morceau", () => {
  expect(MS_PER_DEGREE).toBe(5);
  expect(scratchTarget(60000, 360, 200000)).toBe(61800);
  expect(scratchTarget(60000, -720, 200000)).toBe(56400);
  expect(scratchTarget(1000, -720, 200000)).toBe(0);
  expect(scratchTarget(199000, 360, 200000)).toBe(199000);
});
