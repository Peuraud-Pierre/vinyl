import { expect, test } from "vitest";
import { EQ_BARS, eqStep, eqTempo, newEq, spectrumLevels } from "./equalizer";

// Hasard reproductible pour les tests.
const seeded = (seed: number) => () => ((seed = (seed * 1664525 + 1013904223) >>> 0) / 2 ** 32);

const run = (playing: boolean, ms: number, s = newEq(), rand = seeded(1)) => {
  const seen: number[][] = [];
  for (let t = 0; t < ms; t += 16) { s = eqStep(s, 16, playing, 120, rand); seen.push([...s.levels]); }
  return { s, seen };
};

test("en lecture, les barres bougent et restent entre 0 et 1", () => {
  const { seen } = run(true, 4000);
  const all = seen.flat();
  expect(Math.min(...all)).toBeGreaterThanOrEqual(0);
  expect(Math.max(...all)).toBeLessThanOrEqual(1);
  expect(Math.max(...all)).toBeGreaterThan(0.6);
  expect(seen[seen.length - 1]).not.toEqual(seen[seen.length - 20]);
});

test("les graves montent plus haut que les aigus, en moyenne", () => {
  const { seen } = run(true, 6000);
  const avg = (i: number) => seen.reduce((a, l) => a + l[i]!, 0) / seen.length;
  expect(avg(0)).toBeGreaterThan(avg(EQ_BARS - 1));
});

test("en pause, les barres et les crêtes retombent à zéro, puis l'animation peut s'arrêter", () => {
  const playing = run(true, 2000).s;
  const { s } = run(false, 3000, playing);
  expect(Math.max(...s.levels, ...s.peaks)).toBe(0);
  expect(s.idle).toBe(true);
});

test("tempo stable pour un même morceau, entre 90 et 140 bpm", () => {
  expect(eqTempo("abc")).toBe(eqTempo("abc"));
  for (const id of ["", "a", "spotify:track:1", "12345"]) {
    expect(eqTempo(id)).toBeGreaterThanOrEqual(90);
    expect(eqTempo(id)).toBeLessThan(140);
  }
});

test("vrai son : chaque barre vise la hauteur captée", () => {
  let s = newEq();
  const input = Array.from({ length: EQ_BARS }, (_, i) => (i === 0 ? 1 : 0.1));
  for (let i = 0; i < 30; i++) s = eqStep(s, 16, true, 120, Math.random, input);
  expect(s.levels[0]).toBeGreaterThan(0.95);
  expect(s.levels[EQ_BARS - 1]).toBeCloseTo(0.1, 1);
});

test("spectre : silence reconnu, son fort en haut, gain qui s'adapte", () => {
  const quiet = Array(EQ_BARS).fill(-120);
  expect(spectrumLevels(quiet, -30, 16).silent).toBe(true);
  const loudBass = Array.from({ length: EQ_BARS }, (_, i) => (i < 3 ? -10 : -60));
  const r = spectrumLevels(loudBass, -30, 16);
  expect(r.silent).toBe(false);
  expect(r.levels[0]).toBeGreaterThan(0.8);
  expect(r.levels[EQ_BARS - 1]).toBeLessThan(0.2);
  expect(r.levels.every((v) => v >= 0 && v <= 1)).toBe(true);
  expect(r.gain).toBeGreaterThanOrEqual(-10); // suit le niveau le plus fort
  // Musique plus douce : le gain redescend peu à peu, les barres remontent.
  const soft = loudBass.map((d) => d - 25);
  let g = r.gain, last = 0;
  for (let i = 0; i < 300; i++) { const x = spectrumLevels(soft, g, 16); g = x.gain; last = x.levels[0]!; }
  expect(last).toBeGreaterThan(0.8);
});
