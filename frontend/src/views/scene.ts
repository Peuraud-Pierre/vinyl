// Disposition de la platine : une « pièce » de taille fixe (ROOM, px de scène) mise à l'échelle dans la
// zone disponible, centrée. La table occupe le bas (à partir de TABLE_Y). Sur la table, de gauche à droite :
// le tourne-disque (disque et bras), la pochette sur son support (tournée vers le tourne-disque), la pile. L'étagère est au mur,
// au-dessus de la pile.
import { ARM_GEOMETRY } from "../turntable/arm";
import { STAGE_H } from "../turntable/turntable";
import { SHELF_W } from "../ui/wallshelf";

export const TT_SCALE = 0.75;
export const PILE_SCALE = 0.7;
export const ROOM = { w: 1180, h: 760 };
export const TABLE_Y = 560;
const PILE_H = 460;

export interface Placed { x: number; y: number; scale: number; }
export interface SceneLayout {
  k: number; stage: Placed; pile: Placed; shelf: Placed; tableTop: number;
  sleeve: { x: number; y: number; size: number }; // pochette (px dans la zone), carrée
  stand: { x: number; y: number; w: number; h: number }; // support « en écoute » sous la pochette
}

// Positions dans la pièce (px de scène).
// Tourne-disque : le bord gauche du disque à 10 px du bord (la pochette n'est plus à sa gauche).
const DISC_L = ARM_GEOMETRY.discCenter.x - ARM_GEOMETRY.discRadius;
const STAGE_AT = { x: 10 - DISC_L * TT_SCALE, y: TABLE_Y + 70 - STAGE_H * TT_SCALE }; // le bas sur la table
const PILE_AT = { x: 870, y: TABLE_Y + 40 - PILE_H * PILE_SCALE }; // la base de la pile sur la table, à droite
const SLEEVE_SIZE = 300;
const SLEEVE_AT = { x: 500, y: TABLE_Y + 46 - SLEEVE_SIZE }; // debout sur son support, à gauche de la pile
// Pochette et support tournés de trois quarts vers le tourne-disque (autour du bas de la pochette).
export const SLEEVE_YAW = -42; // degrés (CSS rotateY : négatif = face vers la gauche)
export const SLEEVE_PERSPECTIVE = 1100; // px de scène
// Support « en écoute » (socle en chêne) : un peu plus large que la pochette ; son rebord avant cache le bas
// de la pochette, calée dans la rainure ; posé sur la table.
const STAND = { pad: 18, over: 16, h: 34 };
export const STAND_DEPTH = 70; // profondeur du socle (px de scène), visible sur son côté droit
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
    sleeve: { x: ox + SLEEVE_AT.x * k, y: oy + SLEEVE_AT.y * k, size: SLEEVE_SIZE * k },
    stand: {
      x: ox + (SLEEVE_AT.x - STAND.pad) * k, y: oy + (SLEEVE_AT.y + SLEEVE_SIZE - STAND.over) * k,
      w: (SLEEVE_SIZE + 2 * STAND.pad) * k, h: STAND.h * k,
    },
  };
}
