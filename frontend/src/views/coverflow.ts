// Étagère façon MD Vinyl (vidéo screen/ScreenRecording…, 15–30 s) : tous les boîtiers sont vus
// par la tranche, comme des livres, la pochette en biais tournée vers le centre. Défilement continu.
export const SHELF_GAP = 60; // vidéo : une tranche tous les ~10 % de la largeur // distance entre deux tranches (px, avant mise à l'échelle)
export const SHELF_MARGIN = 55; // marge du premier boîtier par rapport au bord gauche
export const SHELF_ANGLE = 85; // vidéo : près du centre, la pochette ne montre que ~10 % de sa largeur
export const SHELF_PERSPECTIVE = 1650; // px par unité d’échelle : la perspective suit l’agrandissement de la rangée
export const CENTER_GAP = 0; // vidéo : aucun vide au centre (les pochettes partent des tranches vers le centre)

/** Position du boîtier i pour un décalage de défilement donné (x relatif au centre de la scène). */
// Comme dans la vidéo : tranche sur le bord extérieur, face à l'utilisateur ; la pochette part de la
// tranche et s'enfonce vers le centre (à gauche : rotateY positif, tranche à gauche ; à droite : l'inverse).
export function shelfTransform(i: number, offset: number, halfWidth: number): { x: number; rotateY: number; zIndex: number; spineRight: boolean } {
  const base = i * SHELF_GAP - offset - halfWidth + SHELF_MARGIN;
  const onRight = base > 0;
  const x = onRight ? base + CENTER_GAP : base;
  return { x, rotateY: onRight ? -SHELF_ANGLE : SHELF_ANGLE, zIndex: 1000 - Math.round(Math.abs(x)), spineRight: onRight };
}

/** Décalage borné : 0 au début, la fin de la rangée calée sur le bord droit. */
export function clampOffset(offset: number, count: number, halfWidth: number): number {
  const max = Math.max(0, (count - 1) * SHELF_GAP + CENTER_GAP - 2 * (halfWidth - SHELF_MARGIN));
  return Math.min(max, Math.max(0, offset));
}

/** Indices à rendre : ceux dans la scène, plus une tranche de marge de chaque côté. */
export function visibleIndices(offset: number, count: number, halfWidth: number): [number, number] {
  if (count <= 0) return [0, -1];
  const first = Math.floor((offset - SHELF_GAP) / SHELF_GAP);
  const last = Math.ceil((offset + 2 * halfWidth - SHELF_MARGIN - CENTER_GAP + SHELF_GAP) / SHELF_GAP);
  return [Math.max(0, first), Math.min(count - 1, last)];
}
