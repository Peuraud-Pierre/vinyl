import { expect, test } from "vitest";
import { ARM_GEOMETRY } from "../turntable/arm";
import { layoutScene } from "../views/scene";
import { apply, faceMatrices, type Mat4 } from "./pile3d";
import { PILE_TURN, PILE_VIEW } from "./recentpile";
import { mountSleeveStand, SLEEVE_S, sleeveInStage, standGeometry, STAND_BOX } from "./sleevestand";

const corners = (m: Mat4, w: number, h: number) => [[0, 0], [w, 0], [0, h], [w, h]].map(([u, v]) => apply(m, u!, v!));

test("support de pochette : volume en bois (côté, dessus, devant) et « EN ÉCOUTE » gravé", () => {
  const el = document.createElement("div");
  mountSleeveStand(el);
  expect(el.querySelector(".vy-stand-side")).not.toBeNull();
  expect(el.querySelector(".vy-stand-top")).not.toBeNull();
  expect(el.querySelector(".vy-stand-front")!.textContent).toBe("EN ÉCOUTE");
  expect(el.getAttribute("aria-hidden")).toBe("true");
});

// Même caméra que la pile : même point de fuite, même perspective, même rotation sur la table.
test("pochette et support sont projetés comme la pile (vue d'en haut, pivotés comme elle)", () => {
  const g = standGeometry();
  // Vue d'en haut : le dessus du support se voit (il a une hauteur à l'écran), comme la pochette d'un disque.
  const top = corners(g.stand.top, STAND_BOX.w, STAND_BOX.d);
  expect(Math.abs(top[3]![1] - top[1]![1])).toBeGreaterThan(5);
  // Pivoté comme la pile (côté gauche plus loin) : le bord gauche de la pochette paraît plus petit.
  const s = corners(g.sleeve, SLEEVE_S, SLEEVE_S);
  expect(s[2]![1] - s[0]![1]).toBeLessThan(s[3]![1] - s[1]![1]);
  expect(PILE_TURN).toBeLessThan(0);
});

test("ni la pochette ni le support ne passent derrière la pile", () => {
  const g = standGeometry();
  const ours = [
    ...corners(g.sleeve, SLEEVE_S, SLEEVE_S), ...corners(g.stand.front, STAND_BOX.w, STAND_BOX.h),
    ...corners(g.stand.top, STAND_BOX.w, STAND_BOX.d), ...corners(g.stand.side, STAND_BOX.d, STAND_BOX.h),
  ];
  // Disque du bas de la pile, aux extrêmes de son décalage aléatoire (±7 px, ±9°).
  let pileLeft = Infinity;
  for (const yaw of [PILE_TURN - 9, PILE_TURN + 9]) {
    const m = faceMatrices(PILE_VIEW, 0, -7, yaw);
    for (const [x] of [...corners(m.front, PILE_VIEW.width, PILE_VIEW.slab), ...corners(m.top, PILE_VIEW.width, PILE_VIEW.depth)]) pileLeft = Math.min(pileLeft, x!);
  }
  expect(Math.max(...ours.map(([x]) => x!))).toBeLessThan(pileLeft);
});

test("la pochette, dans la platine mise à l'échelle, tombe exactement là où la pile la projette", () => {
  const L = layoutScene(1300, 900);
  const g = standGeometry();
  const inStage = sleeveInStage(g.sleeve, L.pile, L.stage);
  for (const [u, v] of [[0, 0], [SLEEVE_S, 0], [0, SLEEVE_S], [SLEEVE_S, SLEEVE_S]]) {
    const [px, py] = apply(g.sleeve, u!, v!);
    const [sx, sy] = apply(inStage, u!, v!);
    expect(L.stage.x + sx * L.stage.scale).toBeCloseTo(L.pile.x + px * L.pile.scale, 6);
    expect(L.stage.y + sy * L.stage.scale).toBeCloseTo(L.pile.y + py * L.pile.scale, 6);
  }
});

test.each([[1300, 900], [1780, 700], [960, 1000], [600, 400]])("%i×%i : entre le tourne-disque et la pile, posé sur la table", (w, h) => {
  const L = layoutScene(w, h);
  const g = standGeometry();
  const toWrap = ([x, y]: [number, number]) => [L.pile.x + x * L.pile.scale, L.pile.y + y * L.pile.scale];
  const front = corners(g.stand.front, STAND_BOX.w, STAND_BOX.h).map(toWrap);
  const ttRight = L.stage.x + (ARM_GEOMETRY.pivot.x + 50) * L.stage.scale;
  expect(Math.min(...front.map(([x]) => x!))).toBeGreaterThan(ttRight);
  expect(Math.max(...front.map(([, y]) => y!))).toBeGreaterThan(L.tableTop); // le support repose sur la table
});
