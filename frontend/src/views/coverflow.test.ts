import { expect, test } from "vitest";
import { CENTER_GAP, clampOffset, SHELF_ANGLE, SHELF_GAP, SHELF_MARGIN, shelfTransform, smoothStep, visibleIndices } from "./coverflow";

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
test("environ dix tranches sur la largeur (vidéo : une tranche tous les 10 % de l'écran)", () => {
  const visible = (2 * (HALF - SHELF_MARGIN)) / SHELF_GAP;
  expect(visible).toBeGreaterThan(8);
  expect(visible).toBeLessThan(20);
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
  expect(shelfTransform(b, 0, HALF).x).toBeLessThanOrEqual(HALF + 4 * SHELF_GAP);
  expect(shelfTransform(b + 1, 0, HALF).x).toBeGreaterThan(HALF + 3 * SHELF_GAP);
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

test("pochettes presque perpendiculaires : près du centre, à peine 10–15 % de leur hauteur en largeur", () => {
  const visibleRatio = Math.cos((SHELF_ANGLE * Math.PI) / 180);
  expect(visibleRatio).toBeGreaterThan(0.08);
  expect(visibleRatio).toBeLessThan(0.18);
});

test("défilement lissé : la position rattrape la cible en douceur, sans dépasser", () => {
  let x = 0;
  const seen: number[] = [];
  for (let i = 0; i < 30; i++) { x = smoothStep(x, 300, 16); seen.push(x); }
  expect(seen[0]).toBeGreaterThan(0);
  expect(seen[0]).toBeLessThan(120); // pas de saut brutal à la première image
  expect(seen.every((v, i) => i === 0 || v >= seen[i - 1]!)).toBe(true); // monotone
  expect(Math.max(...seen)).toBeLessThanOrEqual(300); // jamais au-delà de la cible
  expect(x).toBe(300); // arrivé (accroché à la fin)
});
test("une longue pause d'image ne fait pas sauter au-delà de la cible", () => {
  expect(smoothStep(0, 100, 5000)).toBe(100);
});

test("passage du centre sans saut : l'angle tend vers 90° des deux côtés (seule la tranche, de face)", () => {
  const near = (dx: number) => shelfTransform(0, -HALF + SHELF_MARGIN - dx, HALF); // boîtier placé à x = dx
  expect(near(-0.01).rotateY).toBeCloseTo(90, 1);
  expect(near(0.01).rotateY).toBeCloseTo(-90, 1);
  // Loin du centre, l'inclinaison normale de l'étagère.
  expect(near(-SHELF_GAP * 2).rotateY).toBe(SHELF_ANGLE);
  expect(near(SHELF_GAP * 2).rotateY).toBe(-SHELF_ANGLE);
});
test("l'angle varie en continu avec la position (pas de saut d'une image à l'autre)", () => {
  let prev = shelfTransform(0, -HALF + SHELF_MARGIN + 200, HALF).rotateY; // x = -200
  for (let x = -199; x < 0; x++) {
    const a = shelfTransform(0, -HALF + SHELF_MARGIN - x, HALF).rotateY;
    expect(Math.abs(a - prev)).toBeLessThan(0.5);
    prev = a;
  }
});
test("les pochettes sont préparées bien avant d'entrer à l'écran (3 boîtiers de marge)", () => {
  const [, b] = visibleIndices(0, 100, HALF);
  expect(shelfTransform(b, 0, HALF).x).toBeGreaterThan(HALF + 2 * SHELF_GAP);
});
