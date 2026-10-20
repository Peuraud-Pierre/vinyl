// Vue de côté de la platine : la platine en volume, vue avec la caméra de la pile (pile3d, même point de
// fuite), dans le même sens que la table (face avant parallèle à son bord). Le dessus de la platine est la vue de dessus actuelle
// (disque, bras, étiquette, inchangés), projetée sur le dessus d'un socle en bois ; les touches passent sur
// sa face avant. Toutes les matrices sont données dans le repère de leur conteneur :
// - wrapper (disque + bras) et deck (touches) : repère de #pl-scaled (px de platine) ;
// - panel (#pl-panel, socle en bois), topFace, front, side : repère de la zone (#pl-wrap).
import { DECK_IN_STAGE, DECK_SIZE, PILE_SCALE, plinthInStage, TT_SCALE, type SceneLayout } from "../views/scene";
import { boxMatrices, mul, scaleXY, translate, type Box, type Mat4 } from "./pile3d";
import { PILE_VIEW } from "./recentpile";
import { STAND_RAIL } from "./sleevestand";

// px de platine → px de pile ; vue de biais, la platine prend plus de place : plus petite (71 %) pour tenir à gauche du support.
const SIDE_SIZE = 0.71;
const SIGMA = (TT_SCALE / PILE_SCALE) * SIDE_SIZE;
const PLINTH = plinthInStage();
const H = 84; // épaisseur du socle (px de pile)
const GAP = 190; // entre la platine et le support de pochette (vue de biais, le dessus part vers la droite)
const W = PLINTH.w * SIGMA, D = PLINTH.h * SIGMA;
/** Socle en volume, dans le repère de la pile, à gauche du support de pochette. */
export const TT_BOX: Box = { x: STAND_RAIL.x - GAP - W, y: PILE_VIEW.height - H, z: 30, w: W, h: H, d: D };
const TT_YAW = 0; // dans le même sens que la table
const PIVOT = { x: TT_BOX.x + W / 2, y: PILE_VIEW.height, z: TT_BOX.z - D / 2 };
// Touches sur la face avant : à gauche, centrées en hauteur.
const DECK_AT = { u: 56, t: (H - DECK_SIZE.h * SIGMA) / 2 };

export interface TurntableView {
  wrapper: Mat4; deck: Mat4; // repère de #pl-scaled
  panel: Mat4; topFace: Mat4; front: Mat4; side: Mat4; // repère de la zone
  frontSize: [number, number]; sideSize: [number, number];
}

const identity: Mat4 = [1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1];

export function sideView(L: SceneLayout): TurntableView {
  const f = boxMatrices(PILE_VIEW, TT_BOX, TT_YAW, PIVOT);
  const toWrap = mul(translate(L.pile.x, L.pile.y, 0), scaleXY(L.pile.scale)); // repère de la pile → zone
  const inStage = (m: Mat4) => mul(scaleXY(1 / L.stage.scale), mul(translate(-L.stage.x, -L.stage.y, 0), m)); // zone → #pl-scaled
  const topFace = mul(toWrap, f.top);
  const stageToTop = mul(scaleXY(SIGMA), translate(-PLINTH.x, -PLINTH.y, 0)); // px de platine → dessus du socle
  const front = mul(toWrap, f.front);
  return {
    wrapper: inStage(mul(topFace, stageToTop)),
    // Socle en bois : px de zone (vue de dessus) → px de platine → dessus du socle.
    panel: mul(topFace, mul(stageToTop, mul(scaleXY(1 / L.stage.scale), translate(L.panel.x - L.stage.x, L.panel.y - L.stage.y, 0)))),
    deck: inStage(mul(front, mul(translate(DECK_AT.u, DECK_AT.t, 0), scaleXY(SIGMA)))),
    topFace, front, side: mul(toWrap, f.side),
    frontSize: [W, H], sideSize: [D, H],
  };
}

/** Vue de dessus : la disposition actuelle, exprimée en matrices (pour la bascule). */
export function topView(L: SceneLayout): TurntableView {
  const s = sideView(L);
  return {
    ...s,
    wrapper: identity,
    panel: translate(L.panel.x, L.panel.y, 0),
    deck: translate(DECK_IN_STAGE.x, DECK_IN_STAGE.y, 0),
  };
}

/** Interpolation des matrices (bascule animée entre les deux vues). */
export function lerpMat(a: Mat4, b: Mat4, t: number): Mat4 {
  return a.map((v, i) => v + (b[i]! - v) * t);
}

/** Ce qui réagit selon la vue (t : 0 = dessus, 1 = côté). En vue de côté, un clic sur la platine ouvre la
 *  vue de dessus (les touches restent actives) ; en vue de dessus, disque et bras réagissent. Le bouton
 *  propose l'autre vue (« top » : passer en vue de dessus, « side » : en vue de côté) ; pendant la
 *  bascule, rien. */
export function viewState(t: number): { topInteractive: boolean; plinthOpensTop: boolean; button: "top" | "side" | null } {
  return { topInteractive: t <= 0, plinthOpensTop: t >= 1, button: t >= 1 ? "top" : t <= 0 ? "side" : null };
}

export const easeInOut = (p: number) => (p < 0.5 ? 2 * p * p : 1 - (-2 * p + 2) ** 2 / 2);
