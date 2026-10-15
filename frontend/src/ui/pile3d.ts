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

const mul = (a: Mat4, b: Mat4): Mat4 => {
  const r = new Array<number>(16).fill(0);
  for (let c = 0; c < 4; c++) for (let row = 0; row < 4; row++) {
    let s = 0;
    for (let k = 0; k < 4; k++) s += a[k * 4 + row]! * b[c * 4 + k]!;
    r[c * 4 + row] = s;
  }
  return r;
};
const translate = (x: number, y: number, z: number): Mat4 => [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, x, y, z, 1];
// Même sens que rotateY() en CSS.
const rotateY = (deg: number): Mat4 => {
  const a = (deg * Math.PI) / 180, c = Math.cos(a), s = Math.sin(a);
  return [c, 0, -s, 0, 0, 1, 0, 0, s, 0, c, 0, 0, 0, 0, 1];
};

/** Matrices des trois faces visibles d'un disque : tranche (avant), pochette (dessus), côté droit. */
export function faceMatrices(v: PileView, level: number, dx: number, yaw: number): { front: Mat4; top: Mat4; side: Mat4 } {
  const yTop = v.height - (level + 1) * v.slab;
  const W = v.width, D = v.depth;
  // Disque : décalé de dx et tourné sur la table autour de son centre.
  const c = { x: W / 2, y: yTop + v.slab / 2, z: -D / 2 };
  const disc = mul(translate(c.x + dx, c.y, c.z), mul(rotateY(yaw), translate(-c.x, -c.y, -c.z)));
  // Perspective centrée sur le point de fuite.
  const persp: Mat4 = [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, -1 / v.perspective, 0, 0, 0, 1];
  const proj = mul(translate(v.eye.x, v.eye.y, 0), mul(persp, translate(-v.eye.x, -v.eye.y, 0)));
  const full = (local: Mat4) => mul(proj, mul(disc, local));
  return {
    // Élément W × slab : (u, t) → (u, yTop + t, 0).
    front: full(translate(0, yTop, 0)),
    // Élément W × D posé à plat : (u, v) → (u, yTop, v − D) ; son bord bas (v = D) touche la tranche.
    top: full([1, 0, 0, 0, 0, 0, 1, 0, 0, -1, 0, 0, 0, yTop, -D, 1]),
    // Élément D × slab : (s, t) → (W, yTop + t, −s) ; son bord gauche (s = 0) touche la tranche.
    side: full([0, 0, -1, 0, 0, 1, 0, 0, 1, 0, 0, 0, W, yTop, 0, 1]),
  };
}

/** Point (u, v) de l'élément → position à l'écran (px de la pile). */
export function apply(m: Mat4, u: number, v: number): [number, number] {
  const x = m[0]! * u + m[4]! * v + m[12]!;
  const y = m[1]! * u + m[5]! * v + m[13]!;
  const w = m[3]! * u + m[7]! * v + m[15]!;
  return [x / w, y / w];
}

export const toCss = (m: Mat4) => `matrix3d(${m.map((n) => +n.toFixed(6)).join(",")})`;
