import { expect, test } from "vitest";
import { slotTransform, visibleRange, WheelStepper } from "./coverflow";

test("élément sélectionné : de face, avancé, au-dessus", () => {
  expect(slotTransform(5, 5)).toEqual({ x: 0, rotateY: 0, z: 120, zIndex: 1000 });
});
test("à gauche : +70°, à droite : −70°, espacés de 64 px après le grand saut", () => {
  const l1 = slotTransform(4, 5), l2 = slotTransform(3, 5), r1 = slotTransform(6, 5);
  expect(l1.rotateY).toBe(70);
  expect(r1.rotateY).toBe(-70);
  expect(l1.x).toBeLessThan(0);
  expect(r1.x).toBeGreaterThan(0);
  expect(l1.x - l2.x).toBe(64);
  expect(l1.zIndex).toBeGreaterThan(l2.zIndex);
});
test("fenêtre ±12 bornée", () => {
  expect(visibleRange(0, 100)).toEqual([0, 12]);
  expect(visibleRange(50, 100)).toEqual([38, 62]);
  expect(visibleRange(98, 100)).toEqual([86, 99]);
  expect(visibleRange(0, 0)).toEqual([0, -1]);
});
test("molette : un cran tous les 60 px, reste conservé", () => {
  const w = new WheelStepper();
  expect(w.push(40)).toBe(0);
  expect(w.push(30)).toBe(1);
  expect(w.push(-130)).toBe(-2);
});
