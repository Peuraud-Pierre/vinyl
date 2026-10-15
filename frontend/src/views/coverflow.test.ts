import { expect, test } from "vitest";
import { CENTER_GAP, clampOffset, SHELF_ANGLE, SHELF_GAP, SHELF_MARGIN, SHELF_SPINE, shelfTransform, springStep, visibleIndices, wheelDelta } from "./coverflow";

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
test("au passage du centre, l'écart entre les deux boîtiers qui l'encadrent tient compte de la tranche", () => {
  // Les deux boîtiers qui encadrent le centre.
  const k = Math.floor((HALF - SHELF_MARGIN) / SHELF_GAP);
  const a = shelfTransform(k, 0, HALF), b = shelfTransform(k + 1, 0, HALF);
  expect(a.x).toBeLessThan(0);
  expect(b.x).toBeGreaterThan(0);
  expect(b.x - a.x).toBe(SHELF_GAP + CENTER_GAP);
});

test("plus de relief (demande utilisateur) : on voit 25 à 40 % de la largeur des pochettes", () => {
  const visibleRatio = Math.cos((SHELF_ANGLE * Math.PI) / 180);
  expect(visibleRatio).toBeGreaterThan(0.25);
  expect(visibleRatio).toBeLessThan(0.4);
});
test("défilement amorti : depuis l'arrêt, la position rejoint la cible en douceur, sans dépasser", () => {
  let st = { x: 0, v: 0 };
  const seen: number[] = [];
  for (let i = 0; i < 40; i++) { st = springStep(st.x, st.v, 300, 16); seen.push(st.x); }
  expect(seen[0]).toBeGreaterThan(0);
  expect(seen[0]).toBeLessThan(120); // pas de saut brutal à la première image
  expect(seen.every((v, i) => i === 0 || v >= seen[i - 1]!)).toBe(true); // monotone
  expect(Math.max(...seen)).toBeLessThanOrEqual(300); // jamais au-delà de la cible
  expect(st).toEqual({ x: 300, v: 0 }); // arrivé et arrêté
});
test("une longue pause d'image ne fait pas sauter au-delà de la cible", () => {
  expect(springStep(0, 0, 100, 5000)).toEqual({ x: 100, v: 0 });
});
test("trackpad (demande utilisateur : saccadé) : vitesse régulière malgré des événements irréguliers", () => {
  // Le trackpad envoie ses événements sans suivre les images : 0, 1 ou 2 par image selon les cas.
  const perFrame = [1, 2, 0, 1, 1, 2, 0, 1, 2, 1, 0, 1, 1, 2, 0, 1, 1, 1, 2, 0];
  const jerk = (stepFn: (x: number, v: number, goal: number) => { x: number; v: number }) => {
    let goal = 0, st = { x: 0, v: 0 }, prevD = 0, sum = 0;
    for (let f = 0; f < 120; f++) {
      goal += 8 * perFrame[f % perFrame.length]!;
      const before = st.x;
      st = stepFn(st.x, st.v, goal);
      const d = st.x - before;
      if (f > 30) sum += Math.abs(d - prevD); // régime établi : variations de vitesse d'une image à l'autre
      prevD = d;
    }
    return sum;
  };
  // Ancien lissage (rattrapage exponentiel simple, constante 70 ms) : la vitesse saute à chaque événement.
  const exponential = (x: number, _v: number, goal: number) => ({ x: x + (goal - x) * (1 - Math.exp(-16 / 70)), v: 0 });
  const spring = (x: number, v: number, goal: number) => springStep(x, v, goal, 16);
  expect(jerk(spring)).toBeLessThan(0.5 * jerk(exponential));
});
test("trackpad : l'axe (horizontal ou vertical) est choisi au début du geste et ne bascule plus ensuite", () => {
  const a = wheelDelta(10, 2, 1000, null); // nouveau geste, surtout horizontal
  expect(a).toEqual({ axis: "x", delta: 10 });
  const b = wheelDelta(1, -3, 16, a.axis); // même geste, petite dérive verticale : on reste à l'horizontale
  expect(b).toEqual({ axis: "x", delta: 1 });
  const c = wheelDelta(0, 12, 400, b.axis); // après une pause : nouveau geste, vertical
  expect(c).toEqual({ axis: "y", delta: 12 });
});

test("passage du centre sans saut : l'angle tend vers 90° des deux côtés (seule la tranche, de face)", () => {
  const at = (dx: number) => shelfTransform(0, -HALF + SHELF_MARGIN - dx, HALF); // boîtier placé à x = dx
  expect(at(-0.01).rotateY).toBeCloseTo(90, 1);
  expect(at(0.01).rotateY).toBeCloseTo(-90, 1);
  // Loin du centre (au-delà de la zone de redressement), l'inclinaison normale.
  expect(at(-SHELF_GAP * 3).rotateY).toBe(SHELF_ANGLE);
  expect(at(SHELF_GAP * 3).rotateY).toBe(-SHELF_ANGLE);
});
test("redressement progressif : courbe douce sur deux boîtiers, sans à-coup de vitesse", () => {
  const angle = (x: number) => Math.abs(shelfTransform(0, -HALF + SHELF_MARGIN - x, HALF).rotateY);
  // Commence à se redresser dès deux boîtiers du centre.
  expect(angle(-SHELF_GAP * 1.5)).toBeGreaterThan(SHELF_ANGLE);
  // Vitesse de rotation (degrés par px) : jamais plus de 2,5 × la moyenne, pas de cassure aux bords de la zone.
  const mean = (90 - SHELF_ANGLE) / (2 * SHELF_GAP);
  let prevRate = 0;
  for (let x = -2 * SHELF_GAP - 5; x < -1; x++) {
    const rate = angle(x + 1) - angle(x);
    expect(rate).toBeGreaterThanOrEqual(0);
    expect(rate).toBeLessThan(2.5 * mean);
    expect(Math.abs(rate - prevRate)).toBeLessThan(0.05);
    prevRate = rate;
  }
});
test("les pochettes sont préparées bien avant d'entrer à l'écran (3 boîtiers de marge)", () => {
  const [, b] = visibleIndices(0, 100, HALF);
  expect(shelfTransform(b, 0, HALF).x).toBeGreaterThan(HALF + 2 * SHELF_GAP);
});

test("le dernier boîtier ne reste jamais caché sous la barre de droite (10 playlists)", () => {
  const half = 333, reserve = 85; // fenêtre 1280 px à l'échelle 1,92 ; barre + fondu ≈ 160 px à l'écran
  const n = 10;
  const max = clampOffset(1e9, n, half, reserve);
  expect(shelfTransform(n - 1, max, half).x).toBeLessThanOrEqual(half - SHELF_MARGIN - reserve + 0.001);
  expect(max).toBeGreaterThan(0); // on peut donc défiler jusqu'à lui
});

test("passage du centre sans saut : la tranche reste au même endroit à l'écran (demande utilisateur)", () => {
  // La tranche est collée au bord extérieur : à gauche elle occupe [x - SPINE, x], à droite [x, x + SPINE].
  // En franchissant le centre, elle change de bord : sans compensation, elle sauterait d'une épaisseur.
  const spineCenter = (dx: number) => {
    const t = shelfTransform(0, -HALF + SHELF_MARGIN - dx, HALF);
    return t.spineRight ? t.x + SHELF_SPINE / 2 : t.x - SHELF_SPINE / 2;
  };
  expect(spineCenter(0.01) - spineCenter(-0.01)).toBeCloseTo(0.02, 3);
});
