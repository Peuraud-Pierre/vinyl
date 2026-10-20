import { expect, test } from "vitest";
import { DECK_SIZE, layoutScene, plinthInStage } from "../views/scene";
import { apply, type Mat4 } from "./pile3d";
import { standGeometry, STAND_RAIL } from "./sleevestand";
import { easeInOut, lerpMat, sideView, topView, TT_BOX, viewState } from "./turntableside";

const SIZES: [number, number][] = [[1300, 900], [1780, 1080], [960, 700], [1780, 700], [600, 400]];
const near = (a: number[], b: number[], d = 4) => { expect(a[0]).toBeCloseTo(b[0]!, d); expect(a[1]).toBeCloseTo(b[1]!, d); };

test.each(SIZES)("%i×%i : en vue de côté, le dessus de la platine est le dessus du socle en volume", (w, h) => {
  const L = layoutScene(w, h);
  const V = sideView(L);
  const r = plinthInStage();
  // Point de la platine (px de platine) → zone, à travers #pl-scaled (position + échelle de la platine).
  const fromStage = (m: Mat4, x: number, y: number) => { const [a, b] = apply(m, x, y); return [L.stage.x + a * L.stage.scale, L.stage.y + b * L.stage.scale]; };
  const corners: [number, number, number, number][] = [[r.x, r.y, 0, 0], [r.x + r.w, r.y, TT_BOX.w, 0], [r.x, r.y + r.h, 0, TT_BOX.d], [r.x + r.w, r.y + r.h, TT_BOX.w, TT_BOX.d]];
  for (const [sx, sy, u, v] of corners) near(fromStage(V.wrapper, sx, sy), apply(V.topFace, u, v));
  // Le socle en bois (#pl-panel, en px de zone) suit le même dessus.
  near(apply(V.panel, 0, 0), apply(V.topFace, 0, 0));
  near(apply(V.panel, L.panel.w, L.panel.h), apply(V.topFace, TT_BOX.w, TT_BOX.d));
});

test.each(SIZES)("%i×%i : touches sur la face avant, platine sans chevaucher le support ni sortir de la zone", (w, h) => {
  const L = layoutScene(w, h);
  const V = sideView(L);
  const fromStage = (m: Mat4, x: number, y: number) => { const [a, b] = apply(m, x, y); return [L.stage.x + a * L.stage.scale, L.stage.y + b * L.stage.scale]; };
  const front = [[0, 0], [V.frontSize[0], 0], [0, V.frontSize[1]], [V.frontSize[0], V.frontSize[1]]].map(([u, v]) => apply(V.front, u!, v!));
  const keys = [[0, 0], [DECK_SIZE.w, 0], [0, DECK_SIZE.h], [DECK_SIZE.w, DECK_SIZE.h]].map(([u, v]) => fromStage(V.deck, u!, v!));
  const xs = front.map((p) => p[0]!), ys = front.map((p) => p[1]!);
  for (const [x, y] of keys) {
    expect(x!).toBeGreaterThan(Math.min(...xs));
    expect(x!).toBeLessThan(Math.max(...xs));
    expect(y!).toBeGreaterThan(Math.min(...ys) - 1);
    expect(y!).toBeLessThan(Math.max(...ys) + 1);
  }
  // Toute la platine (dessus, face avant, côté) dans la zone et à gauche du support.
  const pts = [
    ...[[0, 0], [TT_BOX.w, 0], [0, TT_BOX.d], [TT_BOX.w, TT_BOX.d]].map(([u, v]) => apply(V.topFace, u!, v!)),
    ...front,
    ...[[0, 0], [V.sideSize[0], 0], [0, V.sideSize[1]], [V.sideSize[0], V.sideSize[1]]].map(([u, v]) => apply(V.side, u!, v!)),
  ];
  const g = standGeometry();
  const standLeft = Math.min(...[[0, 0], [0, STAND_RAIL.h]].map(([u, v]) => L.pile.x + apply(g.rail.front, u!, v!)[0]! * L.pile.scale));
  expect(Math.min(...pts.map((p) => p[0]!))).toBeGreaterThanOrEqual(0);
  expect(Math.max(...pts.map((p) => p[0]!))).toBeLessThan(standLeft);
  expect(Math.max(...pts.map((p) => p[1]!))).toBeGreaterThan(L.tableTop); // posée sur la table
});

test("vue de dessus : la disposition actuelle (positions des éléments)", () => {
  const L = layoutScene(1300, 900);
  const T = topView(L);
  near(apply(T.wrapper, 10, 20), [10, 20]);
  near(apply(T.panel, 0, 0), [L.panel.x, L.panel.y]);
});

test("bascule : interpolation des matrices, de la vue de dessus (t = 0) à la vue de côté (t = 1)", () => {
  const a: Mat4 = [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 10, 20, 0, 1], b: Mat4 = [2, 0, 0, 0, 0, 2, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1];
  expect(lerpMat(a, b, 0)).toEqual(a);
  expect(lerpMat(a, b, 1)).toEqual(b);
  near(apply(lerpMat(a, b, 0.5), 0, 0), [5, 10]);
});

// Ce qui réagit selon la vue (t : 0 = dessus, 1 = côté ; entre les deux : bascule en cours).
// Le bouton en haut à gauche bascule d'une vue à l'autre : « vue de dessus » en vue de côté, « vue de
// côté » en vue de dessus ; caché pendant la bascule.
test("interactions selon la vue", () => {
  expect(viewState(1)).toEqual({ topInteractive: false, plinthOpensTop: true, button: "top" });
  expect(viewState(0)).toEqual({ topInteractive: true, plinthOpensTop: false, button: "side" });
  expect(viewState(0.4)).toEqual({ topInteractive: false, plinthOpensTop: false, button: null }); // pendant la bascule
});

test("bascule douce : départ et arrivée en douceur", () => {
  expect(easeInOut(0)).toBe(0);
  expect(easeInOut(1)).toBe(1);
  expect(easeInOut(0.5)).toBeCloseTo(0.5, 6);
  expect(easeInOut(0.1)).toBeLessThan(0.1);
});

// Dans le même sens que la table : face avant parallèle au bord de la table (arête basse horizontale).
test("vue de côté alignée sur la table", () => {
  const L = layoutScene(1300, 900);
  const V = sideView(L);
  const [, yl] = apply(V.front, 0, V.frontSize[1]), [, yr] = apply(V.front, V.frontSize[0], V.frontSize[1]);
  expect(yr).toBeCloseTo(yl, 6);
});
