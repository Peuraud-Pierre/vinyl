import { expect, test } from "vitest";
import { CENTER_GAP, clampOffset, SHELF_GAP, SHELF_MARGIN, shelfTransform, visibleIndices } from "./coverflow";

const HALF = 600; // demi-largeur de la scène

test("au départ, le premier boîtier est contre le bord gauche", () => {
  expect(shelfTransform(0, 0, HALF).x).toBe(-HALF + SHELF_MARGIN);
});
test("défilement continu : les boîtiers glissent du décalage, au pixel près", () => {
  expect(shelfTransform(3, 0, HALF).x - shelfTransform(3, 37, HALF).x).toBe(37);
  expect(shelfTransform(4, 0, HALF).x - shelfTransform(3, 0, HALF).x).toBe(SHELF_GAP);
});
test("tous vus par la tranche : tranche côté centre, face à l'utilisateur ; pochette en biais", () => {
  // Vérifié en capture : à gauche, rotateY négatif (bord droit vers l'avant) et tranche sur le bord droit.
  const left = shelfTransform(0, 0, HALF), right = shelfTransform(12, 0, HALF);
  expect(left.rotateY).toBe(-72);
  expect(left.spineRight).toBe(true);
  expect(right.x).toBeGreaterThan(0);
  expect(right.rotateY).toBe(72);
  expect(right.spineRight).toBe(false);
});
test("les boîtiers proches du centre passent devant leurs voisins", () => {
  const near = shelfTransform(4, 0, HALF), far = shelfTransform(0, 0, HALF);
  expect(Math.abs(near.x)).toBeLessThan(Math.abs(far.x));
  expect(near.zIndex).toBeGreaterThan(far.zIndex);
});
test("bornes du défilement", () => {
  expect(clampOffset(-50, 20, HALF)).toBe(0);
  const max = 19 * SHELF_GAP + CENTER_GAP - 2 * (HALF - SHELF_MARGIN); // le dernier boîtier arrive au bord droit
  expect(clampOffset(99999, 20, HALF)).toBe(max);
  expect(clampOffset(500, 3, HALF)).toBe(0); // peu d'éléments : pas de défilement
});
test("seuls les boîtiers visibles (plus une marge) sont rendus", () => {
  const [a, b] = visibleIndices(0, 100, HALF);
  expect(a).toBe(0);
  expect(shelfTransform(b, 0, HALF).x).toBeLessThanOrEqual(HALF + 2 * SHELF_GAP);
  expect(shelfTransform(b + 1, 0, HALF).x).toBeGreaterThan(HALF + SHELF_GAP);
  expect(visibleIndices(0, 0, HALF)).toEqual([0, -1]);
});
test("au passage du centre, un écart supplémentaire évite que les deux tranches se chevauchent", () => {
  // Le boîtier 4 est à gauche du centre (−94), le 5 à droite (+10 avant écart) : HALF 600, marge 90, pas 104.
  const a = shelfTransform(4, 0, HALF), b = shelfTransform(5, 0, HALF);
  expect(a.x).toBeLessThan(0);
  expect(b.x).toBeGreaterThan(0);
  expect(b.x - a.x).toBe(SHELF_GAP + CENTER_GAP);
});
