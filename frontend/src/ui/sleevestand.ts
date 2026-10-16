// Support de pochette « en écoute » (comme les présentoirs en chêne « Now Playing ») et pochette posée
// dessus, vus avec la même caméra que la pile (pile3d : même point de fuite, même perspective) et pivotés
// comme elle sur la table : tout est décrit dans le repère de la pile, à sa gauche.
//
// Le socle est en deux parties pour l'ordre de dessin : l'arrière (dessus et côté), derrière la pochette ;
// le rebord avant (gravé), devant elle. La pochette est calée dans la rainure entre les deux et penche un
// peu en arrière.
import type { Placed } from "../views/scene";
import { boxMatrices, mul, scaleXY, toCss, translate, type Box, type Faces, type Mat4 } from "./pile3d";
import { PILE_TURN, PILE_VIEW } from "./recentpile";

const BASE = PILE_VIEW.height; // ligne de la table dans le repère de la pile
const LEFT = -572, PAD = 16, H = 48; // socle : bord gauche, débord de chaque côté de la pochette, hauteur
export const SLEEVE_S = 430; // côté de la pochette (px de pile)
export const SLEEVE_TILT = 8; // degrés : la pochette penche en arrière
export const STAND_RAIL: Box = { x: LEFT, y: BASE - H, z: 0, w: SLEEVE_S + 2 * PAD, h: H, d: 14 };
export const STAND_BACK: Box = { x: LEFT, y: BASE - H, z: -14, w: SLEEVE_S + 2 * PAD, h: H, d: 86 };
// Bas de la pochette dans la rainure : sous le dessus du socle, juste derrière le rebord avant.
export const SLEEVE_BOX: Box = { x: LEFT + PAD, y: BASE - 14 - SLEEVE_S, z: -16, w: SLEEVE_S, h: SLEEVE_S, d: 0 };
const PIVOT = { x: LEFT + STAND_RAIL.w / 2, y: BASE, z: -50 };

/** Faces projetées du socle (rebord avant, arrière) et face de la pochette (repère de la pile). */
export function standGeometry(): { rail: Faces; back: Faces; sleeve: Mat4 } {
  return {
    rail: boxMatrices(PILE_VIEW, STAND_RAIL, PILE_TURN, PIVOT),
    back: boxMatrices(PILE_VIEW, STAND_BACK, PILE_TURN, PIVOT),
    sleeve: boxMatrices(PILE_VIEW, SLEEVE_BOX, PILE_TURN, PIVOT, SLEEVE_TILT).front,
  };
}

/** La pochette reste dans la platine (image, clic) : sa matrice, du repère de la pile vers celui de la platine. */
export function sleeveInStage(sleeve: Mat4, pile: Placed, stage: Placed): Mat4 {
  const r = pile.scale / stage.scale;
  return mul(translate((pile.x - stage.x) / stage.scale, (pile.y - stage.y) / stage.scale, 0), mul(scaleXY(r), sleeve));
}

function face(cls: string, w: number, h: number, m: Mat4, text = ""): HTMLElement {
  const f = Object.assign(document.createElement("div"), { className: cls, textContent: text });
  f.style.width = `${w}px`;
  f.style.height = `${h}px`;
  f.style.transform = toCss(m);
  return f;
}

/** Socle (deux calques placés et mis à l'échelle comme la pile) : back derrière la pochette, front devant. */
export function mountSleeveStand(back: HTMLElement, front: HTMLElement): void {
  const g = standGeometry();
  for (const el of [back, front]) {
    el.classList.add("vy-stand");
    el.setAttribute("aria-hidden", "true");
  }
  back.replaceChildren(
    face("vy-stand-side", STAND_BACK.d, STAND_BACK.h, g.back.side),
    face("vy-stand-top vy-stand-groove", STAND_BACK.w, STAND_BACK.d, g.back.top),
  );
  front.replaceChildren(
    face("vy-stand-side", STAND_RAIL.d, STAND_RAIL.h, g.rail.side),
    face("vy-stand-top", STAND_RAIL.w, STAND_RAIL.d, g.rail.top),
    face("vy-stand-front", STAND_RAIL.w, STAND_RAIL.h, g.rail.front, "EN ÉCOUTE"),
  );
}
