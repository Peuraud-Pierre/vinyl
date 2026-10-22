// Mode DJ : on attrape le disque et on le fait tourner à la main. Un tour vaut ce que dure un tour à
// 33⅓ tours/min (1,8 s) : tourner en avant avance dans le morceau, en arrière revient en arrière ; tant
// qu'on tient le disque, le morceau n'avance plus (comme un plateau retenu).
export const MS_PER_DEGREE = 1800 / 360;

/** Angle du pointeur autour du centre du disque (degrés, 0 = en haut, sens des aiguilles d'une montre). */
export const pointerAngle = (cx: number, cy: number, x: number, y: number) => (Math.atan2(x - cx, -(y - cy)) * 180) / Math.PI;

/** Écart d'angle le plus court de a à b (degrés, entre -180 et 180) : pas de saut en passant le haut. */
export const turnDelta = (a: number, b: number) => ((b - a + 540) % 360) - 180;

/** Position visée (ms) après avoir tourné le disque de « turnedDeg » depuis la position « fromMs ». */
export function scratchTarget(fromMs: number, turnedDeg: number, durationMs: number): number {
  const p = fromMs + turnedDeg * MS_PER_DEGREE;
  return Math.round(Math.max(0, durationMs > 0 ? Math.min(p, durationMs - 1000) : p));
}
