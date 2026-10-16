// Disposition de la platine : une « pièce » de taille fixe (ROOM, px de scène) mise à l'échelle dans la
// zone disponible, centrée. La table occupe le bas (à partir de TABLE_Y) ; platine et pile, réduites, y
// reposent ; l'étagère est au mur, au-dessus de la pile.
import { STAGE_H } from "../turntable/turntable";
import { SHELF_W } from "../ui/wallshelf";

export const TT_SCALE = 0.75;
export const PILE_SCALE = 0.7;
export const ROOM = { w: 1180, h: 760 };
export const TABLE_Y = 560;
const PILE_H = 460;

export interface Placed { x: number; y: number; scale: number; }
export interface SceneLayout { k: number; stage: Placed; pile: Placed; shelf: Placed; tableTop: number; }

// Positions dans la pièce (px de scène).
const STAGE_AT = { x: 10, y: TABLE_Y + 70 - STAGE_H * TT_SCALE }; // le bas de la platine sur la table
const PILE_AT = { x: 780, y: TABLE_Y + 40 - PILE_H * PILE_SCALE }; // la base de la pile sur la table
const SHELF_AT = { x: ROOM.w - SHELF_W - 30, y: 90 };

export function layoutScene(w: number, h: number): SceneLayout {
  const k = Math.min(w / ROOM.w, h / ROOM.h);
  const ox = (w - ROOM.w * k) / 2, oy = (h - ROOM.h * k) / 2;
  const at = (p: { x: number; y: number }, scale: number): Placed => ({ x: ox + p.x * k, y: oy + p.y * k, scale });
  return {
    k,
    stage: at(STAGE_AT, k * TT_SCALE),
    pile: at(PILE_AT, k * PILE_SCALE),
    shelf: at(SHELF_AT, k),
    tableTop: oy + TABLE_Y * k,
  };
}
