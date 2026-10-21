// Petit égaliseur rétro posé sur le socle de la platine : boîtier bois et métal, rangée de barres
// lumineuses (couleur d'accent) avec leurs crêtes. Sur Mac, il suit le vrai son : Vinyl capte la sortie
// audio et reçoit l'énergie de 12 bandes de fréquences (événement « spectrum »). Sans capture (Windows,
// autorisation refusée, réglage désactivé), les barres sont simulées sur un tempo propre au morceau.

export const EQ_BARS = 12;

export interface EqState {
  levels: number[]; // hauteur de chaque barre (0 à 1), des graves aux aigus
  peaks: number[]; // crête de chaque barre, qui retombe lentement
  drift: number[]; // part aléatoire lissée de chaque barre
  t: number; // temps écoulé (ms), pour le battement
  idle: boolean; // tout est retombé : plus rien à dessiner
}

export const newEq = (): EqState => ({
  levels: Array(EQ_BARS).fill(0), peaks: Array(EQ_BARS).fill(0), drift: Array(EQ_BARS).fill(0), t: 0, idle: true,
});

/** Tempo du morceau (bpm) : toujours le même pour un même identifiant. */
export function eqTempo(trackId: string): number {
  let h = 2166136261;
  for (let i = 0; i < trackId.length; i++) h = Math.imul(h ^ trackId.charCodeAt(i), 16777619) >>> 0;
  return 90 + (h % 50);
}

const clamp = (v: number) => Math.min(1, Math.max(0, v));

/** Hauteurs (0 à 1) tirées des bandes captées (dB). Les aigus, naturellement plus faibles, sont relevés ;
 *  le gain suit le passage le plus fort et redescend doucement (6 dB/s) pour les morceaux plus calmes. */
export function spectrumLevels(db: number[], gain: number, dt: number): { levels: number[]; gain: number; silent: boolean } {
  const tilted = db.map((d, i) => d + i * 2.5);
  const loudest = Math.max(...tilted);
  const g = Math.max(loudest, gain - dt * 0.006, -60);
  const RANGE = 40;
  return {
    levels: tilted.map((d) => clamp((d - (g - RANGE)) / RANGE) ** 1.5),
    gain: g,
    silent: Math.max(...db) < -85,
  };
}

/** Une image de l'animation : dt en ms ; input, s'il est fourni, donne la hauteur visée de chaque barre. */
export function eqStep(s: EqState, dt: number, playing: boolean, bpm: number, rand: () => number = Math.random, input?: number[]): EqState {
  const t = s.t + dt;
  const phase = ((t / 60000) * bpm) % 1;
  const kick = Math.exp(-phase * 7); // coup de grosse caisse à chaque temps
  const off = Math.exp(-((phase + 0.5) % 1) * 9); // contretemps (charleston), surtout dans les aigus
  const levels: number[] = [], peaks: number[] = [], drift: number[] = [];
  for (let i = 0; i < EQ_BARS; i++) {
    const f = i / (EQ_BARS - 1); // 0 : graves, 1 : aigus
    const d = s.drift[i]! * 0.9 + (rand() - 0.5) * 0.5;
    drift.push(d);
    const simulated = clamp(0.5 - 0.22 * f + kick * (1 - f) * 0.45 + off * f * 0.3 + d * 0.35);
    const target = playing ? (input ? clamp(input[i] ?? 0) : simulated) : 0;
    const cur = s.levels[i]!;
    // Montée rapide, descente plus douce (comme une aiguille de VU-mètre).
    const level = target > cur ? cur + (target - cur) * 0.55 : Math.max(target, cur - dt * 0.0022);
    levels.push(level);
    peaks.push(Math.max(level, s.peaks[i]! - dt * 0.0006));
  }
  return { levels, peaks, drift, t, idle: !playing && levels.every((v) => v === 0) && peaks.every((v) => v <= 0) };
}

export interface Equalizer {
  setPlaying(playing: boolean, trackId: string): void;
  setLive(on: boolean): void; // suivre le vrai son quand c'est possible
  setScale(k: number): void;
  destroy(): void;
}

/** Capture du son (backend) : lancée seulement pendant la lecture. */
export interface EqSource {
  start(): Promise<unknown>;
  stop(): Promise<unknown>;
  onSpectrum(cb: (db: number[]) => void): () => void;
}

export function mountEqualizer(root: HTMLElement, source?: EqSource): Equalizer {
  root.innerHTML = `
    <div class="vy-eq" aria-hidden="true">
      <i class="vy-eq-screw"></i><i class="vy-eq-screw"></i>
      <div class="vy-eq-glass">${Array.from({ length: EQ_BARS }, () => `<span class="vy-eq-bar"><b></b><i></i></span>`).join("")}</div>
    </div>`;
  const box = root.querySelector<HTMLElement>(".vy-eq")!;
  const bars = Array.from(root.querySelectorAll<HTMLElement>(".vy-eq-bar b"));
  const caps = Array.from(root.querySelectorAll<HTMLElement>(".vy-eq-bar i"));
  const still = matchMedia("(prefers-reduced-motion: reduce)");
  let state = newEq();
  let playing = false, bpm = 120, raf = 0, last = 0;
  // Vrai son : dernières hauteurs captées, heure de réception, et « déjà entendu » (une capture muette,
  // autorisation refusée, laisse la simulation).
  let live = false, capturing = false, heard = false, gain = -30, lastSpectrum = 0, lastDb = 0;
  let input: number[] | undefined;
  const offSpectrum = source?.onSpectrum((db) => {
    const now = performance.now();
    const r = spectrumLevels(db, gain, lastDb ? now - lastDb : 16);
    lastDb = lastSpectrum = now;
    gain = r.gain;
    if (!r.silent) heard = true;
    input = r.levels;
  });
  const capture = () => {
    const want = !!source && live && playing;
    if (want === capturing) return;
    capturing = want;
    (want ? source!.start() : source!.stop()).catch(() => { capturing = false; });
  };

  const paint = () => state.levels.forEach((v, i) => {
    bars[i]!.style.transform = `scaleY(${v})`;
    caps[i]!.style.transform = `translateY(${-state.peaks[i]! * 100}%)`;
  });
  const frame = (now: number) => {
    const dt = Math.min(50, last ? now - last : 16);
    last = now;
    const fresh = heard && now - lastSpectrum < 500;
    state = eqStep(state, dt, playing && !still.matches, bpm, Math.random, fresh ? input : undefined);
    paint();
    raf = state.idle ? 0 : requestAnimationFrame(frame);
  };
  const wake = () => { if (!raf) { last = 0; raf = requestAnimationFrame(frame); } };
  paint();
  return {
    setPlaying(on, trackId) {
      playing = on;
      bpm = eqTempo(trackId);
      capture();
      if (on || !state.idle) wake();
    },
    setLive(on) { live = on; capture(); },
    // zoom plutôt qu'une mise à l'échelle : le boîtier est dessiné à sa taille d'affichage, donc net.
    setScale(k) { box.style.zoom = String(k); },
    destroy() {
      cancelAnimationFrame(raf);
      raf = 0;
      offSpectrum?.();
      if (capturing) source!.stop().catch(() => {});
    },
  };
}
