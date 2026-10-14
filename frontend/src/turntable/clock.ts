export function estimatePosition(positionMs: number, durationMs: number, playing: boolean, receivedAt: number, now: number): number {
  if (!playing) return positionMs;
  const p = positionMs + Math.max(0, now - receivedAt);
  return durationMs > 0 ? Math.min(p, durationMs) : p;
}
