import { expect, test } from "vitest";
import { apply, faceMatrices, type PileView } from "./pile3d";

const view: PileView = { width: 500, height: 460, depth: 500, slab: 20, eye: { x: 200, y: -70 }, perspective: 1800 };
const near = (a: [number, number], b: [number, number]) => {
  expect(a[0]).toBeCloseTo(b[0], 6);
  expect(a[1]).toBeCloseTo(b[1], 6);
};

test("sans rotation, la tranche du bas est à sa place (pas de déformation à z = 0)", () => {
  const m = faceMatrices(view, 0, 0, 0);
  near(apply(m.front, 0, 0), [0, 440]);
  near(apply(m.front, 500, 20), [500, 460]);
});

test("les faces d'un disque se rejoignent, même tourné : pochette posée sur la tranche, côté contre la tranche", () => {
  for (const [level, dx, yaw] of [[0, 0, 0], [3, 6, -27], [6, -7, -11]] as const) {
    const m = faceMatrices(view, level, dx, yaw);
    // Bord avant de la pochette (v = profondeur) = bord haut de la tranche.
    near(apply(m.top, 0, 500), apply(m.front, 0, 0));
    near(apply(m.top, 500, 500), apply(m.front, 500, 0));
    // Bord avant du côté (s = 0) = bord droit de la tranche.
    near(apply(m.side, 0, 0), apply(m.front, 500, 0));
    near(apply(m.side, 0, 20), apply(m.front, 500, 20));
  }
});

test("vue d'en haut : la pochette posée à plat est visible (elle a une hauteur à l'écran)", () => {
  const m = faceMatrices(view, 6, 0, -20);
  const front = apply(m.top, 250, 500)[1], back = apply(m.top, 250, 0)[1];
  expect(front - back).toBeGreaterThan(40); // le fond de la pochette remonte à l'écran
});

test("plus loin = plus petit (perspective)", () => {
  const m = faceMatrices(view, 0, 0, 0);
  const frontW = apply(m.top, 500, 500)[0] - apply(m.top, 0, 500)[0];
  const backW = apply(m.top, 500, 0)[0] - apply(m.top, 0, 0)[0];
  expect(backW).toBeLessThan(frontW);
});
