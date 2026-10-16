import { expect, test } from "vitest";
import { STAGE_H, STAGE_W } from "../turntable/turntable";
import { SHELF_H, SHELF_W } from "../ui/wallshelf";
import { layoutScene, PILE_SCALE, TT_SCALE } from "./scene";

const PILE = { w: 500, h: 460 };
// Tailles de #pl-wrap (fenêtre moins la colonne de commandes de 140 px), y compris les extrêmes.
const SIZES: [number, number][] = [[1300, 900], [1780, 1080], [960, 700], [1780, 700], [960, 1000], [600, 400]];

test.each(SIZES)("%i×%i : tout tient dans la zone, posé sur la table", (w, h) => {
  const L = layoutScene(w, h);
  const right = (p: { x: number; scale: number }, width: number) => p.x + width * p.scale;
  // Rien ne déborde à droite (sous la colonne de commandes) ni à gauche.
  for (const [p, width] of [[L.stage, STAGE_W], [L.pile, PILE.w], [L.shelf, SHELF_W]] as const) {
    expect(p.x).toBeGreaterThanOrEqual(0);
    expect(right(p, width)).toBeLessThanOrEqual(w + 0.5);
  }
  expect(L.stage.scale).toBeCloseTo(L.k * TT_SCALE, 6);
  expect(L.pile.scale).toBeCloseTo(L.k * PILE_SCALE, 6);
  // Platine et pile reposent sur la table : leur bas passe sous la ligne de la table.
  expect(L.stage.y + STAGE_H * L.stage.scale).toBeGreaterThan(L.tableTop);
  expect(L.pile.y + PILE.h * L.pile.scale).toBeGreaterThan(L.tableTop);
  expect(L.tableTop).toBeLessThan(h);
  // L'étagère est au mur : au-dessus de la table et de la pile, à droite de la platine.
  expect(L.shelf.y + SHELF_H * L.shelf.scale).toBeLessThan(L.tableTop);
  expect(L.shelf.x).toBeGreaterThan(L.stage.x + STAGE_W * L.stage.scale * 0.6);
});

test("la scène est centrée et grandit avec la fenêtre", () => {
  const small = layoutScene(960, 700), big = layoutScene(1780, 1080);
  expect(big.k).toBeGreaterThan(small.k);
  const L = layoutScene(1780, 700); // très large : marges égales à gauche et à droite de la scène
  const left = L.stage.x, rightEdge = L.shelf.x + SHELF_W * L.shelf.scale;
  expect(Math.abs(left - (1780 - rightEdge))).toBeLessThan(200);
});
