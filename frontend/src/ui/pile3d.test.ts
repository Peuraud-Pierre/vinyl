import { expect, test } from "vitest";
import { apply, atScale, boxMatrices, faceMatrices, type PileView } from "./pile3d";

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

// Pavé quelconque (plateau de la table, pieds) : mêmes raccords que pour un disque.
test("les faces d'un pavé se rejoignent, même tourné", () => {
  const box = { x: -40, y: 300, z: 30, w: 580, h: 22, d: 560 };
  const m = boxMatrices(view, box, -20, { x: 250, y: 300, z: -250 });
  near(apply(m.top, 0, box.d), apply(m.front, 0, 0));
  near(apply(m.top, box.w, box.d), apply(m.front, box.w, 0));
  near(apply(m.side, 0, 0), apply(m.front, box.w, 0));
  near(apply(m.side, 0, box.h), apply(m.front, box.w, box.h));
});

test("un disque est un pavé comme un autre (faceMatrices = boxMatrices)", () => {
  const a = faceMatrices(view, 2, 5, -15);
  const yTop = view.height - 3 * view.slab;
  const b = boxMatrices(view, { x: 5, y: yTop, z: 0, w: 500, h: 20, d: 500 }, -15, { x: 255, y: yTop + 10, z: -250 });
  near(apply(a.front, 120, 7), apply(b.front, 120, 7));
  near(apply(a.top, 33, 400), apply(b.top, 33, 400));
});

// Pavé penché en arrière (tilt, en degrés) autour de son arête avant-basse : le haut recule (z négatif),
// le bas ne bouge pas.
test("un pavé penché en arrière garde son arête basse, son haut recule", () => {
  const box = { x: 0, y: 0, z: 0, w: 100, h: 200, d: 0 };
  const straight = boxMatrices(view, box, 0, { x: 50, y: 200, z: 0 });
  const leaning = boxMatrices(view, box, 0, { x: 50, y: 200, z: 0 }, 10);
  near(apply(leaning.front, 0, 200), apply(straight.front, 0, 200));
  near(apply(leaning.front, 100, 200), apply(straight.front, 100, 200));
  // Vu d'en haut, un haut qui recule paraît plus haut à l'écran… et le pavé paraît plus court.
  expect(apply(leaning.front, 50, 0)[1]).toBeGreaterThan(apply(straight.front, 50, 0)[1]);
});

// Dessin à la taille d'affichage : un élément s fois plus grand, avec la matrice atScale(m, s), tombe
// exactement là où m l'aurait mis puis agrandi de s — mais il est rendu net (pas d'agrandissement).
test("atScale : même projection, élément dessiné à la taille d'affichage", () => {
  const m = faceMatrices(view, 3, 4, -20).top;
  for (const s of [0.6, 1, 1.8]) {
    const big = atScale(m, s);
    for (const [u, v] of [[0, 0], [500, 0], [0, 500], [250, 120]]) {
      const [x, y] = apply(m, u!, v!);
      near(apply(big, u! * s, v! * s), [x * s, y * s]);
    }
  }
});
