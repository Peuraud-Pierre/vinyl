import { expect, test } from "vitest";
import { CENTER_GAP, clampOffset, SHELF_ANGLE, SHELF_GAP, SHELF_MARGIN, shelfTransform, visibleIndices } from "./coverflow";

const HALF = 600; // demi-largeur de la scène

test("au départ, le premier boîtier est contre le bord gauche", () => {
  expect(shelfTransform(0, 0, HALF).x).toBe(-HALF + SHELF_MARGIN);
});
test("défilement continu : les boîtiers glissent du décalage, au pixel près", () => {
  expect(shelfTransform(3, 0, HALF).x - shelfTransform(3, 37, HALF).x).toBe(37);
  expect(shelfTransform(4, 0, HALF).x - shelfTransform(3, 0, HALF).x).toBe(SHELF_GAP);
});
test("comme dans la vidéo : tranche sur le bord extérieur, pochette qui part vers le centre", () => {
  // Vidéo (16–18 s) : à gauche, la tranche « Colors - Single Egofear » est à gauche et la pochette s'étend
  // vers la droite ; à droite, la tranche « Amy Winehouse » est à droite et la pochette s'étend vers la gauche.
  const left = shelfTransform(0, 0, HALF), right = shelfTransform(14, 0, HALF);
  expect(left.rotateY).toBe(SHELF_ANGLE);
  expect(left.spineRight).toBe(false);
  expect(right.x).toBeGreaterThan(0);
  expect(right.rotateY).toBe(-SHELF_ANGLE);
  expect(right.spineRight).toBe(true);
});
test("cinq à dix boîtiers sur la largeur", () => {
  const visible = (2 * (HALF - SHELF_MARGIN)) / SHELF_GAP;
  expect(visible).toBeGreaterThan(4);
  expect(visible).toBeLessThan(11);
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
  // Les deux boîtiers qui encadrent le centre.
  const k = Math.floor((HALF - SHELF_MARGIN) / SHELF_GAP);
  const a = shelfTransform(k, 0, HALF), b = shelfTransform(k + 1, 0, HALF);
  expect(a.x).toBeLessThan(0);
  expect(b.x).toBeGreaterThan(0);
  expect(b.x - a.x).toBe(SHELF_GAP + CENTER_GAP);
});

test("pochettes moins tournées (demande utilisateur) : on en voit 60 à 75 % de la largeur", () => {
  const visibleRatio = Math.cos((SHELF_ANGLE * Math.PI) / 180);
  expect(visibleRatio).toBeGreaterThan(0.6);
  expect(visibleRatio).toBeLessThan(0.75);
});
test("boîtiers bien séparés : l'écart dépasse la largeur visible d'une pochette (pas de chevauchement)", () => {
  const visibleWidth = 300 * Math.cos((SHELF_ANGLE * Math.PI) / 180);
  expect(SHELF_GAP).toBeGreaterThan(visibleWidth);
});
