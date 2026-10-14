// Angles du bras en degrés. 0° = bras vertical ; positif = rotation horaire, vers le disque.
export const ARM_REST = 4;
export const ARM_START = 18;
export const ARM_END = 34;
export const ARM_LIFT = 12;

const clamp = (v: number, lo: number, hi: number) => Math.min(hi, Math.max(lo, v));

export function angleForProgress(positionMs: number, durationMs: number): number {
  if (!(durationMs > 0) || !Number.isFinite(positionMs)) return ARM_START;
  const p = clamp(positionMs / durationMs, 0, 1);
  return ARM_START + (ARM_END - ARM_START) * p;
}

export function positionForAngle(angle: number, durationMs: number): number {
  if (!(durationMs > 0)) return 0;
  const p = (clamp(angle, ARM_START, ARM_END) - ARM_START) / (ARM_END - ARM_START);
  return Math.round(p * durationMs);
}

export type Release = { kind: "pause" } | { kind: "seek"; positionMs: number };

export function releaseAction(angle: number, durationMs: number): Release {
  if (angle < ARM_LIFT) return { kind: "pause" };
  return { kind: "seek", positionMs: positionForAngle(angle, durationMs) };
}

export function angleFromPointer(pivot: { x: number; y: number }, pointer: { x: number; y: number }): number {
  const dx = pointer.x - pivot.x;
  const dy = pointer.y - pivot.y;
  return (Math.atan2(-dx, dy) * 180) / Math.PI;
}

// Géométrie de la scène de la platine (960×660 px, coordonnées depuis le coin haut-gauche) :
// pochette à gauche, disque au centre, bras vertical à droite.
export const ARM_GEOMETRY = {
  pivot: { x: 880, y: 70 },
  length: 460, // du pivot à la pointe de l'aiguille
  discCenter: { x: 630, y: 340 },
  discRadius: 224,
  labelRadius: 224 * 0.45,
};

const PIVOT_DEADZONE = 30; // px autour du pivot où un clic ne déplace pas le bras
export const CLICK_TOLERANCE = 4; // px de déplacement en dessous desquels un geste est un clic

// dragAngleFromPointer donne l'angle du bras pour un pointeur, ou null si le geste doit être ignoré
// (clic sur la tête du pivot, pointeur au-dessus du pivot).
export function dragAngleFromPointer(pivot: { x: number; y: number }, pointer: { x: number; y: number }): number | null {
  if (Math.hypot(pointer.x - pivot.x, pointer.y - pivot.y) < PIVOT_DEADZONE) return null;
  const a = angleFromPointer(pivot, pointer);
  if (Math.abs(a) > 90) return null;
  return clamp(a, ARM_REST, 40);
}

export type ArmAction =
  | { kind: "none" }
  | { kind: "play" }
  | { kind: "pause" }
  | { kind: "seek"; positionMs: number; resume: boolean };

// armRelease décide de l'action quand on relâche le bras.
export function armRelease(o: { endAngle: number; movedPx: number; playing: boolean; durationMs: number }): ArmAction {
  if (o.movedPx < CLICK_TOLERANCE) return o.playing ? { kind: "none" } : { kind: "play" };
  const r = releaseAction(o.endAngle, o.durationMs);
  if (r.kind === "pause") return o.playing ? { kind: "pause" } : { kind: "none" };
  if (!(o.durationMs > 0)) return o.playing ? { kind: "none" } : { kind: "play" };
  return { kind: "seek", positionMs: r.positionMs, resume: !o.playing };
}
