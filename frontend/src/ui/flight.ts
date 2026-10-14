// Vol de la pochette entre l'étagère et la page album (vidéo MD Vinyl, 20–21 s).
export interface Rect { left: number; top: number; width: number; height: number; }

// Doit correspondre à la mise en page de views/album.ts (padding gauche 92 px, haut 110 + 24 px).
export const COVER_LEFT = 92;
export const COVER_TOP = 134;
export const coverSize = (viewportW: number) => Math.round(Math.min(viewportW * 0.36, 560) * 10) / 10;

export function albumCoverRect(viewportW: number): { left: number; top: number; size: number } {
  return { left: COVER_LEFT, top: COVER_TOP, size: coverSize(viewportW) };
}

/** Images clés : du boîtier (vu en biais, angle rotateY) jusqu'à la pochette à plat sur la page. */
export function flightFrames(from: Rect, to: { left: number; top: number; size: number }, angle: number): Keyframe[] {
  return [
    { left: `${from.left}px`, top: `${from.top}px`, width: `${from.width}px`, height: `${from.height}px`, transform: `perspective(1300px) rotateY(${angle}deg)` },
    { left: `${to.left}px`, top: `${to.top}px`, width: `${to.size}px`, height: `${to.size}px`, transform: "none" },
  ];
}
