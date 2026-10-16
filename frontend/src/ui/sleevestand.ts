// Support de pochette « en écoute » (comme les présentoirs en chêne « Now Playing ») et pochette posée
// dessus, vus avec la même caméra que la pile (pile3d : même point de fuite, même perspective) et pivotés
// comme elle sur la table : tout est décrit dans le repère de la pile, à sa gauche.
import type { Placed } from "../views/scene";
import { boxMatrices, mul, scaleXY, toCss, translate, type Box, type Faces, type Mat4 } from "./pile3d";
import { PILE_TURN, PILE_VIEW } from "./recentpile";

const BASE = PILE_VIEW.height; // ligne de la table dans le repère de la pile
export const SLEEVE_S = 430; // côté de la pochette (px de pile)
// Pochette debout, un peu en retrait du rebord avant (elle est calée dans la rainure).
export const SLEEVE_BOX: Box = { x: -560, y: BASE - SLEEVE_S, z: -34, w: SLEEVE_S, h: SLEEVE_S, d: 0 };
// Socle : un peu plus large que la pochette ; rebord avant, dessus (avec la rainure) et côté droit.
export const STAND_BOX: Box = { x: -586, y: BASE - 48, z: 0, w: SLEEVE_S + 52, h: 48, d: 100 };
const PIVOT = { x: STAND_BOX.x + STAND_BOX.w / 2, y: BASE, z: -STAND_BOX.d / 2 };

/** Faces projetées du socle et face de la pochette (repère de la pile). */
export function standGeometry(): { stand: Faces; sleeve: Mat4 } {
  return {
    stand: boxMatrices(PILE_VIEW, STAND_BOX, PILE_TURN, PIVOT),
    sleeve: boxMatrices(PILE_VIEW, SLEEVE_BOX, PILE_TURN, PIVOT).front,
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

/** Socle (à placer et mettre à l'échelle comme la pile) : côté, dessus, puis rebord avant gravé. */
export function mountSleeveStand(el: HTMLElement): void {
  el.classList.add("vy-stand");
  el.setAttribute("aria-hidden", "true");
  const { stand } = standGeometry();
  el.replaceChildren(
    face("vy-stand-side", STAND_BOX.d, STAND_BOX.h, stand.side),
    face("vy-stand-top", STAND_BOX.w, STAND_BOX.d, stand.top),
    face("vy-stand-front", STAND_BOX.w, STAND_BOX.h, stand.front, "EN ÉCOUTE"),
  );
}
