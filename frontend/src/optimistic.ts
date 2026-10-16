// Lecture/pause instantanée : au clic, l'état inverse est affiché sans attendre la réponse de Musique ou
// Spotify (~0,3 s). Les états reçus entre-temps qui le contredisent sont ignorés, sauf changement de morceau
// ou délai dépassé (commande sans effet : l'état réel revient).
import type { Snapshot } from "./api";

const HOLD_MS = 1500;

export function createOptimistic(now: () => number = Date.now) {
  let pending: { trackId: string; state: string; until: number } | null = null;
  return {
    toggle(s: Snapshot | undefined): Snapshot | null {
      if (!s || s.problem || !s.nowPlaying.trackId) return null;
      const state = s.nowPlaying.state === "playing" ? "paused" : "playing";
      pending = { trackId: s.nowPlaying.trackId, state, until: now() + HOLD_MS };
      return { ...s, nowPlaying: { ...s.nowPlaying, state } };
    },
    filter(s: Snapshot): Snapshot {
      if (!pending) return s;
      if (s.problem || s.nowPlaying.trackId !== pending.trackId || now() >= pending.until || s.nowPlaying.state === pending.state) {
        pending = null;
        return s;
      }
      return { ...s, nowPlaying: { ...s.nowPlaying, state: pending.state } };
    },
    cancel() { pending = null; },
  };
}
