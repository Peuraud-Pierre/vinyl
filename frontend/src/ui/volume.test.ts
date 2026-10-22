import { expect, test } from "vitest";
import { angleDelta, knobAngle, latestOnly, stepVolume, turnVolume } from "./volume";

test("volume : un envoi à la fois, seule la dernière valeur demandée pendant l'envoi suit", async () => {
  const sent: number[] = [];
  let release: () => void = () => {};
  const send = latestOnly((v) => { sent.push(v); return new Promise<void>((r) => (release = r)); });
  send(10); send(20); send(30); send(40);
  expect(sent).toEqual([10]);
  release();
  await new Promise((r) => setTimeout(r, 0));
  expect(sent).toEqual([10, 40]);
  release();
  await new Promise((r) => setTimeout(r, 0));
  expect(sent).toEqual([10, 40]);
});

test("molette : pas de 5, borné entre 0 et 100", () => {
  expect(stepVolume(50, -3)).toBe(55);
  expect(stepVolume(50, 3)).toBe(45);
  expect(stepVolume(98, -1)).toBe(100);
  expect(stepVolume(2, 1)).toBe(0);
});

test("molette : de -135° à +135°, tourne par écart d'angle sans sauter d'un bout à l'autre", () => {
  expect(knobAngle(0)).toBe(-135);
  expect(knobAngle(50)).toBe(0);
  expect(knobAngle(100)).toBe(135);
  expect(angleDelta(170, -170)).toBe(20);
  expect(angleDelta(-170, 170)).toBe(-20);
  expect(turnVolume(50, 0, 27)).toBeCloseTo(60);
  expect(turnVolume(50, 10, -17)).toBeCloseTo(40);
  expect(turnVolume(98, 0, 90)).toBe(100);
  expect(turnVolume(2, 0, -90)).toBe(0);
});
