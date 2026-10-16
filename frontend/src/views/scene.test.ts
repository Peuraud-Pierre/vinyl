import { expect, test } from "vitest";
import { ARM_GEOMETRY } from "../turntable/arm";
import { STAGE_H } from "../turntable/turntable";
import { SHELF_H, SHELF_W } from "../ui/wallshelf";
import { layoutScene, PILE_SCALE, TT_SCALE } from "./scene";

const PILE = { w: 500, h: 460 };
const CONTROLS_W = 140; // colonne de commandes : libre en bas, la pile peut y déborder un peu
const DISC_L = ARM_GEOMETRY.discCenter.x - ARM_GEOMETRY.discRadius; // bord gauche du disque (px de platine)
// Tailles de #pl-wrap (fenêtre moins la colonne de commandes de 140 px), y compris les extrêmes.
const SIZES: [number, number][] = [[1300, 900], [1780, 1080], [960, 700], [1780, 700], [960, 1000], [600, 400]];

test.each(SIZES)("%i×%i : tout tient dans la zone, posé sur la table", (w, h) => {
  const L = layoutScene(w, h);
  // Tourne-disque à gauche (sa pochette est déplacée) : son disque est visible, rien ne déborde à gauche.
  expect(L.stage.x + DISC_L * L.stage.scale).toBeGreaterThanOrEqual(0);
  // L'étagère ne passe pas sous la colonne de commandes ; la pile peut y déborder un peu (bas libre).
  expect(L.shelf.x + SHELF_W * L.shelf.scale).toBeLessThanOrEqual(w + 0.5);
  expect(L.pile.x + PILE.w * L.pile.scale).toBeLessThanOrEqual(w + CONTROLS_W * 0.6);
  expect(L.stage.scale).toBeCloseTo(L.k * TT_SCALE, 6);
  expect(L.pile.scale).toBeCloseTo(L.k * PILE_SCALE, 6);
  // Platine et pile reposent sur la table : leur bas passe sous la ligne de la table.
  expect(L.stage.y + STAGE_H * L.stage.scale).toBeGreaterThan(L.tableTop);
  expect(L.pile.y + PILE.h * L.pile.scale).toBeGreaterThan(L.tableTop);
  expect(L.tableTop).toBeLessThan(h);
  // L'étagère est au mur : au-dessus de la table.
  expect(L.shelf.y + SHELF_H * L.shelf.scale).toBeLessThan(L.tableTop);
});

test("la scène est centrée et grandit avec la fenêtre", () => {
  const small = layoutScene(960, 700), big = layoutScene(1780, 1080);
  expect(big.k).toBeGreaterThan(small.k);
  const L = layoutScene(1780, 700); // très large : marges égales à gauche et à droite de la scène
  const left = L.stage.x + DISC_L * L.stage.scale, rightEdge = L.shelf.x + SHELF_W * L.shelf.scale;
  expect(Math.abs(left - (1780 - rightEdge))).toBeLessThan(200);
});

test("la pile est poussée vers la droite, sous l'étagère", () => {
  for (const [w, h] of SIZES) {
    const L = layoutScene(w, h);
    expect(L.pile.x).toBeGreaterThan(L.shelf.x); // commence à droite du début de l'étagère
  }
});
