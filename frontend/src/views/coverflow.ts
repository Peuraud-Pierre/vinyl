// Coverflow façon MD Vinyl : sélection de face au centre, voisins inclinés à ±70° montrant leur tranche.
export const SPINE_GAP = 64;
const FIRST_GAP = 230; // écart entre la sélection et le premier voisin
export const RADIUS = 12;

export function slotTransform(i: number, focus: number): { x: number; rotateY: number; z: number; zIndex: number } {
  const d = i - focus;
  if (d === 0) return { x: 0, rotateY: 0, z: 120, zIndex: 1000 };
  const side = Math.sign(d);
  const x = side * (FIRST_GAP + (Math.abs(d) - 1) * SPINE_GAP);
  return { x, rotateY: -side * 70, z: 0, zIndex: 1000 - Math.abs(d) };
}

export function visibleRange(focus: number, count: number, radius = RADIUS): [number, number] {
  if (count <= 0) return [0, -1];
  return [Math.max(0, focus - radius), Math.min(count - 1, focus + radius)];
}

// Accumule les deltas de molette ou de trackpad : un cran tous les `step` px, le reste est conservé.
export class WheelStepper {
  private acc = 0;
  constructor(private step = 60) {}
  push(delta: number): number {
    this.acc += delta;
    const n = Math.trunc(this.acc / this.step);
    this.acc -= n * this.step;
    return n;
  }
}
