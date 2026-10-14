import { expect, test } from "vitest";
import { FULL_SPEED, Spinner } from "./spin";

test("accélère jusqu'à 33⅓ tours/min en 600 ms", () => {
  const s = new Spinner();
  s.step(300, true);
  expect(s.speed).toBeCloseTo(FULL_SPEED / 2);
  s.step(300, true);
  expect(s.speed).toBeCloseTo(FULL_SPEED);
  s.step(1000, true);
  expect(s.speed).toBeCloseTo(FULL_SPEED);
});

test("un tour complet prend 1800 ms à pleine vitesse", () => {
  const s = new Spinner();
  s.speed = FULL_SPEED;
  s.step(1800, true);
  expect(s.angle % 360).toBeCloseTo(0);
});

test("ralentit jusqu'à l'arrêt en 1200 ms", () => {
  const s = new Spinner();
  s.speed = FULL_SPEED;
  s.step(600, false);
  expect(s.speed).toBeCloseTo(FULL_SPEED / 2);
  s.step(600, false);
  expect(s.speed).toBe(0);
  const a = s.angle;
  s.step(500, false);
  expect(s.angle).toBe(a);
});

test("l'angle reste dans [0, 360)", () => {
  const s = new Spinner();
  s.speed = FULL_SPEED;
  for (let i = 0; i < 100; i++) s.step(97, true);
  expect(s.angle).toBeGreaterThanOrEqual(0);
  expect(s.angle).toBeLessThan(360);
});
