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
  // Près du centre, l'angle monte en continu jusqu'à 90° : à gauche comme à droite, on ne voit plus
  // que la tranche, de face, au même endroit. Le changement de côté est donc invisible (aucun saut).
  const ramp = Math.min(1, Math.abs(base) / SHELF_GAP);
  const angle = 90 - (90 - SHELF_ANGLE) * ramp;
  return { x, rotateY: onRight ? -angle : angle, zIndex: 1000 - Math.round(Math.abs(x)), spineRight: onRight };
}

/**
 * Décalage borné : 0 au début ; à la fin, le dernier boîtier s'arrête avant la zone réservée à
 * droite (barre de commandes et fondu), pour qu'il ne reste jamais caché dessous.
 */
export function clampOffset(offset: number, count: number, halfWidth: number, rightReserve = 0): number {
  const max = Math.max(0, (count - 1) * SHELF_GAP + CENTER_GAP - 2 * (halfWidth - SHELF_MARGIN) + rightReserve);
  return Math.min(max, Math.max(0, offset));
}

/** Indices à rendre : ceux dans la scène, plus trois boîtiers de marge de chaque côté. */
export function visibleIndices(offset: number, count: number, halfWidth: number): [number, number] {
  if (count <= 0) return [0, -1];
  // Trois boîtiers de marge de chaque côté : leurs pochettes sont décodées avant d'entrer à l'écran.
  const margin = 3 * SHELF_GAP;
  const first = Math.floor((offset - margin) / SHELF_GAP);
  const last = Math.ceil((offset + 2 * halfWidth - SHELF_MARGIN - CENTER_GAP + margin) / SHELF_GAP);
  return [Math.max(0, first), Math.min(count - 1, last)];
}

/**
 * Défilement lissé, calculé image par image : la position rattrape la cible de façon exponentielle
 * (constante de temps tau), sans jamais la dépasser, et s'y accroche à moins d'un demi-pixel.
 */
export function smoothStep(current: number, target: number, dtMs: number, tau = 70): number {
  const k = 1 - Math.exp(-Math.max(0, dtMs) / tau);
  const next = current + (target - current) * k;
  return Math.abs(target - next) < 0.5 ? target : next;
}
