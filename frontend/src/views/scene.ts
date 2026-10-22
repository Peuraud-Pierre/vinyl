// Disposition de la platine : une « pièce » de taille fixe (ROOM, px de scène) mise à l'échelle dans la
// zone disponible, centrée. La table occupe le bas (à partir de TABLE_Y). Sur la table, de gauche à droite :
// le tourne-disque (disque et bras), la pile. L'étagère est au mur,
// au-dessus de la pile.
import { ARM_GEOMETRY } from "../turntable/arm";
import { STAGE_H } from "../turntable/turntable";
import { LYRICS_W } from "../ui/lyricsboard";
import { SHELF_W } from "../ui/wallshelf";

export const TT_SCALE = 0.75;
export const PILE_SCALE = 0.7;
export const ROOM = { w: 1180, h: 760 };
export const TABLE_Y = 560;
const PILE_H = 460;

export interface Placed { x: number; y: number; scale: number; }
export interface SceneLayout {
  k: number; stage: Placed; pile: Placed; shelf: Placed; lyrics: Placed; knob: Placed; tableTop: number;
  panel: { x: number; y: number; w: number; h: number }; // socle de la platine (bois foncé), sur la table
}

// Positions dans la pièce (px de scène).
// Tourne-disque : le bord gauche du disque à 40 px du bord (socle compris) (la pochette n'est plus à sa gauche).
const DISC_L = ARM_GEOMETRY.discCenter.x - ARM_GEOMETRY.discRadius;
const STAGE_AT = { x: 40 - DISC_L * TT_SCALE, y: TABLE_Y + 70 - STAGE_H * TT_SCALE }; // le bas sur la table
const PILE_AT = { x: 870, y: TABLE_Y + 40 - PILE_H * PILE_SCALE }; // la base de la pile sur la table, à droite
// Socle de la platine (bois foncé, vu du dessus comme le disque) : sous le disque, le bras et les touches,
// posé sur la table (il dépasse de sa ligne).
const PANEL_AT = { x: 16, y: 120, w: 444, bottom: 640 };
// Touches précédent / aléatoire / suivant sur le socle, en bas à gauche (px de platine) ; taille du bloc.
export const DECK_IN_STAGE = { x: 420, y: 600 };
export const DECK_SIZE = { w: 202, h: 42 };
// Égaliseur rétro (ui/equalizer), posé sur le socle à droite des touches (px de platine).
export const EQ_IN_STAGE = { x: 660, y: 594 };
const SHELF_AT = { x: ROOM.w - SHELF_W - 30, y: 90 };
// Tableau des paroles (ui/lyricsboard) au mur, centré entre le socle de la platine et l'étagère.
// Molette de volume (ui/volume) : son centre, en haut à gauche du socle de la platine.
export const KNOB_SIZE = 84;
const KNOB_AT = { x: PANEL_AT.x + 58, y: PANEL_AT.y + 56 };
const LYRICS_AT = { x: (PANEL_AT.x + PANEL_AT.w + SHELF_AT.x - LYRICS_W) / 2, y: 64 };

export function layoutScene(w: number, h: number): SceneLayout {
  const k = Math.min(w / ROOM.w, h / ROOM.h);
  const ox = (w - ROOM.w * k) / 2, oy = (h - ROOM.h * k) / 2;
  const at = (p: { x: number; y: number }, scale: number): Placed => ({ x: ox + p.x * k, y: oy + p.y * k, scale });
  return {
    k,
    stage: at(STAGE_AT, k * TT_SCALE),
    pile: at(PILE_AT, k * PILE_SCALE),
    shelf: at(SHELF_AT, k),
    lyrics: at(LYRICS_AT, k),
    knob: at(KNOB_AT, k),
    tableTop: oy + TABLE_Y * k,
    panel: { x: ox + PANEL_AT.x * k, y: oy + PANEL_AT.y * k, w: PANEL_AT.w * k, h: (PANEL_AT.bottom - PANEL_AT.y) * k },
  };
}
