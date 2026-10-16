// Projection en perspective des faces d'un disque de la pile, calculée ici plutôt que laissée à la 3D
// du navigateur (preserve-3d) : dans l'app (WebKit), la pile s'aplatissait après coup. Chaque face
// reçoit une matrice complète (perspective comprise), comme un élément isolé : rendu toujours correct.
// Repère : x vers la droite, y vers le bas, z vers l'observateur ; px de la pile.

export type Mat4 = number[]; // 4 × 4, colonne par colonne (ordre de matrix3d)

export interface PileView {
  width: number; height: number; // boîte de la pile (px)
  depth: number; // profondeur d'une pochette (carrée : = width)
  slab: number; // épaisseur d'une tranche
  eye: { x: number; y: number }; // point de fuite (au-dessus de la pile : vue un peu d'en haut)
  perspective: number; // distance de l'observateur
}

export const mul = (a: Mat4, b: Mat4): Mat4 => {
  const r = new Array<number>(16).fill(0);
  for (let c = 0; c < 4; c++) for (let row = 0; row < 4; row++) {
    let s = 0;
    for (let k = 0; k < 4; k++) s += a[k * 4 + row]! * b[c * 4 + k]!;
    r[c * 4 + row] = s;
  }
  return r;
};
export const translate = (x: number, y: number, z: number): Mat4 => [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, x, y, z, 1];
export const scaleXY = (k: number): Mat4 => [k, 0, 0, 0, 0, k, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1];
// Même sens que rotateY() en CSS.
const rotateY = (deg: number): Mat4 => {
  const a = (deg * Math.PI) / 180, c = Math.cos(a), s = Math.sin(a);
  return [c, 0, -s, 0, 0, 1, 0, 0, s, 0, c, 0, 0, 0, 0, 1];
};

/** Pavé : coin avant-haut-gauche (x, y, z), largeur w, hauteur h, profondeur d (vers −z). */
export interface Box { x: number; y: number; z: number; w: number; h: number; d: number; }
export type Faces = { front: Mat4; top: Mat4; side: Mat4 };

// Même sens que rotateX() en CSS : un angle positif fait reculer (z négatif) ce qui est au-dessus de l'axe.
const rotateX = (deg: number): Mat4 => {
  const a = (deg * Math.PI) / 180, c = Math.cos(a), s = Math.sin(a);
  return [1, 0, 0, 0, 0, c, s, 0, 0, -s, c, 0, 0, 0, 0, 1];
};

/** Matrices des trois faces visibles d'un pavé tourné de yaw autour de pivot : face avant, dessus, côté droit.
 *  tilt (degrés) : le pavé penche en arrière autour de son arête avant-basse (pochette sur un support). */
export function boxMatrices(v: PileView, b: Box, yaw: number, pivot: { x: number; y: number; z: number }, tilt = 0): Faces {
  const lean = mul(translate(0, b.y + b.h, b.z), mul(rotateX(tilt), translate(0, -(b.y + b.h), -b.z)));
  const turn = mul(translate(pivot.x, pivot.y, pivot.z), mul(rotateY(yaw), mul(translate(-pivot.x, -pivot.y, -pivot.z), lean)));
  // Perspective centrée sur le point de fuite.
  const persp: Mat4 = [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, -1 / v.perspective, 0, 0, 0, 1];
  const proj = mul(translate(v.eye.x, v.eye.y, 0), mul(persp, translate(-v.eye.x, -v.eye.y, 0)));
  const full = (local: Mat4) => mul(proj, mul(turn, local));
  return {
    // Élément w × h : (u, t) → (x + u, y + t, z).
    front: full(translate(b.x, b.y, b.z)),
    // Élément w × d posé à plat : (u, v) → (x + u, y, z − d + v) ; son bord bas (v = d) touche la face avant.
    top: full([1, 0, 0, 0, 0, 0, 1, 0, 0, -1, 0, 0, b.x, b.y, b.z - b.d, 1]),
    // Élément d × h : (s, t) → (x + w, y + t, z − s) ; son bord gauche (s = 0) touche la face avant.
    side: full([0, 0, -1, 0, 0, 1, 0, 0, 1, 0, 0, 0, b.x + b.w, b.y, b.z, 1]),
  };
}

/** Matrices des trois faces visibles d'un disque : tranche (avant), pochette (dessus), côté droit. */
export function faceMatrices(v: PileView, level: number, dx: number, yaw: number): Faces {
  const yTop = v.height - (level + 1) * v.slab;
  // Disque : décalé de dx et tourné sur la table autour de son centre.
  return boxMatrices(v, { x: dx, y: yTop, z: 0, w: v.width, h: v.slab, d: v.depth }, yaw,
    { x: v.width / 2 + dx, y: yTop + v.slab / 2, z: -v.depth / 2 });
}

/** Point (u, v) de l'élément → position à l'écran (px de la pile). */
export function apply(m: Mat4, u: number, v: number): [number, number] {
  const x = m[0]! * u + m[4]! * v + m[12]!;
  const y = m[1]! * u + m[5]! * v + m[13]!;
  const w = m[3]! * u + m[7]! * v + m[15]!;
  return [x / w, y / w];
}

export const toCss = (m: Mat4) => `matrix3d(${m.map((n) => +n.toFixed(6)).join(",")})`;
