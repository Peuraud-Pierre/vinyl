import { expect, test } from "vitest";
import { ARM_GEOMETRY } from "../turntable/arm";
import { STAGE_H } from "../turntable/turntable";
import { LYRICS_H, LYRICS_W } from "../ui/lyricsboard";
import { SHELF_H, SHELF_W } from "../ui/wallshelf";
import { DECK_IN_STAGE, DECK_SIZE, KNOB_SIZE, layoutScene, PILE_SCALE, TT_SCALE } from "./scene";

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

// Socle de la platine (bois foncé) : sous le disque et le bras seulement (pas tout le mur), posé sur la
// table, avec les touches dessus ; il s'arrête avant la pochette.
test.each(SIZES)("%i×%i : socle de platine sous le disque, le bras et les touches", (w, h) => {
  const L = layoutScene(w, h);
  const s = L.stage.scale, G = ARM_GEOMETRY;
  const disc = { l: L.stage.x + (G.discCenter.x - G.discRadius) * s, r: L.stage.x + (G.discCenter.x + G.discRadius) * s, t: L.stage.y + (G.discCenter.y - G.discRadius) * s };
  const armR = L.stage.x + (G.pivot.x + 48) * s, armT = L.stage.y + (G.pivot.y - 48) * s;
  const keys = { l: L.stage.x + DECK_IN_STAGE.x * s, r: L.stage.x + (DECK_IN_STAGE.x + DECK_SIZE.w) * s,
    t: L.stage.y + DECK_IN_STAGE.y * s, b: L.stage.y + (DECK_IN_STAGE.y + DECK_SIZE.h) * s };
  const P = L.panel;
  expect(P.x).toBeLessThanOrEqual(disc.l);
  expect(P.x + P.w).toBeGreaterThanOrEqual(Math.max(disc.r, armR));
  expect(P.y).toBeLessThanOrEqual(Math.min(disc.t, armT));
  // Touches sur le socle, sous le disque.
  expect(keys.l).toBeGreaterThan(P.x);
  expect(keys.r).toBeLessThan(P.x + P.w);
  expect(keys.t).toBeGreaterThan(disc.t);
  expect(keys.b).toBeLessThan(P.y + P.h);
  // Posé sur la table (il dépasse de sa ligne), avant la pochette, pas tout le mur.
  expect(P.y + P.h).toBeGreaterThan(L.tableTop);
  expect(P.x + P.w).toBeLessThan(L.pile.x - 300 * L.pile.scale);
  expect(P.w).toBeLessThan(w * 0.6);
});

test.each(SIZES)("%i×%i : le tableau des paroles est au mur, entre la platine et l'étagère", (w, h) => {
  const L = layoutScene(w, h);
  const right = L.lyrics.x + LYRICS_W * L.lyrics.scale, bottom = L.lyrics.y + LYRICS_H * L.lyrics.scale;
  expect(L.lyrics.x).toBeGreaterThan(L.panel.x + L.panel.w);
  expect(right).toBeLessThan(L.shelf.x);
  expect(L.lyrics.y).toBeGreaterThanOrEqual(36 * L.k); // un peu d'air au-dessus
  expect(bottom).toBeLessThan(L.tableTop - 40 * L.k); // au mur, au-dessus de la table
});

test.each(SIZES)("%i×%i : la molette de volume est en haut à gauche du socle, hors du disque", (w, h) => {
  const L = layoutScene(w, h);
  const r = (KNOB_SIZE / 2) * L.knob.scale;
  expect(L.knob.x - r).toBeGreaterThan(L.panel.x);
  expect(L.knob.y - r).toBeGreaterThan(L.panel.y);
  // Le disque : centre et rayon de la platine mise à l'échelle.
  const cx = L.stage.x + ARM_GEOMETRY.discCenter.x * L.stage.scale, cy = L.stage.y + ARM_GEOMETRY.discCenter.y * L.stage.scale;
  expect(Math.hypot(L.knob.x - cx, L.knob.y - cy)).toBeGreaterThan(ARM_GEOMETRY.discRadius * L.stage.scale + r);
});
