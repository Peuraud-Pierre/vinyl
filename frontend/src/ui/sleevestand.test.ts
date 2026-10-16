import { expect, test } from "vitest";
import { ARM_GEOMETRY } from "../turntable/arm";
import { layoutScene } from "../views/scene";
import { apply, faceMatrices, type Mat4 } from "./pile3d";
import { PILE_TURN, PILE_VIEW } from "./recentpile";
import { mountSleeveStand, SLEEVE_BOX, SLEEVE_S, SLEEVE_TILT, sleeveInStage, standGeometry, STAND_BACK, STAND_RAIL } from "./sleevestand";

const corners = (m: Mat4, w: number, h: number) => [[0, 0], [w, 0], [0, h], [w, h]].map(([u, v]) => apply(m, u!, v!));

test("support de pochette : arrière (dessus, côté) derrière la pochette, rebord avant gravé devant", () => {
  const back = document.createElement("div"), front = document.createElement("div");
  mountSleeveStand(back, front);
  expect(back.querySelector(".vy-stand-top")).not.toBeNull();
  expect(back.querySelector(".vy-stand-side")).not.toBeNull();
  expect(front.querySelector(".vy-stand-front")!.textContent).toBe("EN ÉCOUTE");
  expect(front.querySelector(".vy-stand-top")).not.toBeNull(); // bord du rebord, devant la pochette
  for (const el of [back, front]) expect(el.getAttribute("aria-hidden")).toBe("true");
});

// La pochette est calée dans la rainure : son bas est sous le dessus du socle, juste derrière le rebord
// avant, et elle penche un peu en arrière comme sur un vrai support.
test("la pochette est calée dans la rainure, centrée, et penche en arrière", () => {
  const bottom = SLEEVE_BOX.y + SLEEVE_BOX.h;
  expect(bottom).toBeGreaterThan(STAND_RAIL.y); // sous le dessus du socle (cachée par le rebord)
  expect(bottom).toBeLessThan(STAND_RAIL.y + STAND_RAIL.h);
  expect(SLEEVE_BOX.z).toBeLessThanOrEqual(STAND_RAIL.z - STAND_RAIL.d); // juste derrière le rebord
  expect(SLEEVE_BOX.z).toBeGreaterThan(STAND_BACK.z - STAND_BACK.d);
  const mid = (b: { x: number; w: number }) => b.x + b.w / 2;
  expect(mid(SLEEVE_BOX)).toBeCloseTo(mid(STAND_RAIL), 6);
  expect(SLEEVE_TILT).toBeGreaterThan(0);
});

// Même caméra que la pile : même point de fuite, même perspective, même rotation sur la table.
test("pochette et support sont projetés comme la pile (vue d'en haut, pivotés comme elle)", () => {
  const g = standGeometry();
  const top = corners(g.back.top, STAND_BACK.w, STAND_BACK.d);
  expect(Math.abs(top[3]![1] - top[1]![1])).toBeGreaterThan(5); // le dessus se voit
  const s = corners(g.sleeve, SLEEVE_S, SLEEVE_S);
  expect(s[2]![1] - s[0]![1]).toBeLessThan(s[3]![1] - s[1]![1]); // bord gauche plus loin, plus petit
  expect(PILE_TURN).toBeLessThan(0);
});

test("ni la pochette ni le support ne passent derrière la pile", () => {
  const g = standGeometry();
  const ours = [
    ...corners(g.sleeve, SLEEVE_S, SLEEVE_S),
    ...[g.rail, g.back].flatMap((f, i) => {
      const b = i === 0 ? STAND_RAIL : STAND_BACK;
      return [...corners(f.front, b.w, b.h), ...corners(f.top, b.w, b.d), ...corners(f.side, b.d, b.h)];
    }),
  ];
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
  const front = corners(g.rail.front, STAND_RAIL.w, STAND_RAIL.h).map(toWrap);
  const ttRight = L.stage.x + (ARM_GEOMETRY.pivot.x + 50) * L.stage.scale;
  expect(Math.min(...front.map(([x]) => x!))).toBeGreaterThan(ttRight);
  expect(Math.max(...front.map(([, y]) => y!))).toBeGreaterThan(L.tableTop); // le support repose sur la table
});
