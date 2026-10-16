import { expect, test } from "vitest";
import { createOptimistic } from "./optimistic";
import type { Snapshot } from "./api";

const snap = (state: string, trackId = "A"): Snapshot => ({
  nowPlaying: { source: "applemusic", trackId, title: "T", artist: "X", album: "", durationMs: 1000, positionMs: 0, state, artworkUrl: "" },
  problem: "", problemApp: "",
});

test("clic : l'état inverse est affiché tout de suite", () => {
  const o = createOptimistic(() => 0);
  expect(o.toggle(snap("playing"))?.nowPlaying.state).toBe("paused");
  expect(o.toggle(snap("paused"))?.nowPlaying.state).toBe("playing");
});

test("pas de lecteur : rien à inverser", () => {
  const o = createOptimistic(() => 0);
  expect(o.toggle({ ...snap("playing"), problem: "no-player" })).toBeNull();
});

test("un ancien état arrivé avant la confirmation ne fait pas clignoter le disque", () => {
  let t = 0;
  const o = createOptimistic(() => t);
  o.toggle(snap("playing")); // → pause affichée
  t = 200;
  expect(o.filter(snap("playing")).nowPlaying.state).toBe("paused"); // ancien état : ignoré
  t = 300;
  expect(o.filter(snap("paused")).nowPlaying.state).toBe("paused"); // confirmé
  t = 400;
  expect(o.filter(snap("playing")).nowPlaying.state).toBe("playing"); // relancé ailleurs : suivi
});

test("commande sans effet : l'état réel revient au bout d'un moment", () => {
  let t = 0;
  const o = createOptimistic(() => t);
  o.toggle(snap("playing"));
  t = 2000;
  expect(o.filter(snap("playing")).nowPlaying.state).toBe("playing");
});

test("autre morceau : suivi aussitôt", () => {
  const o = createOptimistic(() => 0);
  o.toggle(snap("playing"));
  expect(o.filter(snap("playing", "B")).nowPlaying.trackId).toBe("B");
  expect(o.filter(snap("playing", "B")).nowPlaying.state).toBe("playing");
});

test("échec de la commande : l'attente est annulée", () => {
  const o = createOptimistic(() => 0);
  o.toggle(snap("playing"));
  o.cancel();
  expect(o.filter(snap("playing")).nowPlaying.state).toBe("playing");
});
