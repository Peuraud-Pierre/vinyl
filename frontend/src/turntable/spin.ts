export const FULL_SPEED = 360 / 1800; // degrés par ms (33⅓ tours/min)
const ACCEL = FULL_SPEED / 600;
const DECEL = FULL_SPEED / 1200;

export class Spinner {
  angle = 0;
  speed = 0;

  step(dtMs: number, playing: boolean): number {
    if (playing) this.speed = Math.min(FULL_SPEED, this.speed + ACCEL * dtMs);
    else this.speed = Math.max(0, this.speed - DECEL * dtMs);
    this.angle = (this.angle + this.speed * dtMs) % 360;
    return this.angle;
  }
}
