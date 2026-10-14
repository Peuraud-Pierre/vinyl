import { describe, expect, test } from "vitest";
import { ARM_END, ARM_GEOMETRY, ARM_REST, ARM_START, armRelease, dragAngleFromPointer, angleForProgress, angleFromPointer, positionForAngle, releaseAction } from "./arm";

describe("bras", () => {
  test("début, milieu et fin du morceau", () => {
    expect(angleForProgress(0, 200_000)).toBe(ARM_START);
    expect(angleForProgress(100_000, 200_000)).toBe((ARM_START + ARM_END) / 2);
    expect(angleForProgress(200_000, 200_000)).toBe(ARM_END);
  });

  test("durée inconnue : pas de NaN, l'aiguille reste au début", () => {
    expect(angleForProgress(5000, 0)).toBe(ARM_START);
    expect(Number.isNaN(angleForProgress(NaN, NaN))).toBe(false);
    expect(positionForAngle(26, 0)).toBe(0);
  });

  test("valeurs hors bornes bornées", () => {
    expect(angleForProgress(-10, 1000)).toBe(ARM_START);
    expect(angleForProgress(5000, 1000)).toBe(ARM_END);
  });

  test("angle → position", () => {
    expect(positionForAngle(ARM_START, 200_000)).toBe(0);
    expect(positionForAngle(ARM_END, 200_000)).toBe(200_000);
    expect(positionForAngle(40, 200_000)).toBe(200_000);
  });

  test("relâcher le bras", () => {
    expect(releaseAction(ARM_REST, 1000)).toEqual({ kind: "pause" });
    expect(releaseAction(11.9, 1000)).toEqual({ kind: "pause" });
    expect(releaseAction(14, 1000)).toEqual({ kind: "seek", positionMs: 0 });
    expect(releaseAction(26, 1000)).toEqual({ kind: "seek", positionMs: 500 });
  });

  test("angle depuis le pointeur (0° = vertical, positif vers la gauche)", () => {
    const pivot = { x: 100, y: 0 };
    expect(angleFromPointer(pivot, { x: 100, y: 50 })).toBeCloseTo(0);
    expect(angleFromPointer(pivot, { x: 50, y: 50 })).toBeCloseTo(45);
    expect(angleFromPointer(pivot, { x: 150, y: 50 })).toBeCloseTo(-45);
  });
});

describe("géométrie du bras sur la platine", () => {
  const dist = (angle: number) => {
    const t = (angle * Math.PI) / 180;
    const x = ARM_GEOMETRY.pivot.x - ARM_GEOMETRY.length * Math.sin(t);
    const y = ARM_GEOMETRY.pivot.y + ARM_GEOMETRY.length * Math.cos(t);
    return Math.hypot(x - ARM_GEOMETRY.discCenter.x, y - ARM_GEOMETRY.discCenter.y);
  };
  test("au repos, l'aiguille est hors du disque", () => {
    expect(dist(ARM_REST)).toBeGreaterThan(ARM_GEOMETRY.discRadius);
  });
  test("début du morceau : aiguille sur le bord du disque", () => {
    expect(dist(ARM_START)).toBeLessThan(ARM_GEOMETRY.discRadius);
    expect(dist(ARM_START)).toBeGreaterThan(ARM_GEOMETRY.discRadius * 0.8);
  });
  test("fin du morceau : aiguille près de l'étiquette, sans la toucher", () => {
    expect(dist(ARM_END)).toBeGreaterThan(ARM_GEOMETRY.labelRadius);
    expect(dist(ARM_END)).toBeLessThan(ARM_GEOMETRY.labelRadius * 1.5);
  });
});

describe("geste sur le bras", () => {
  const pivot = { x: 100, y: 100 };
  test("clic sur la tête du pivot : ignoré", () => {
    expect(dragAngleFromPointer(pivot, { x: 100, y: 85 })).toBeNull(); // juste au-dessus du pivot
    expect(dragAngleFromPointer(pivot, { x: 110, y: 110 })).toBeNull(); // à moins de 30 px
  });
  test("pointeur au-dessus du pivot (angle > 90°) : ignoré", () => {
    expect(dragAngleFromPointer(pivot, { x: 60, y: 40 })).toBeNull();
  });
  test("pointeur sur le corps du bras : angle borné", () => {
    expect(dragAngleFromPointer(pivot, { x: 100, y: 200 })).toBeCloseTo(ARM_REST); // vertical → borné au repos
    expect(dragAngleFromPointer(pivot, { x: 0, y: 200 })).toBeCloseTo(40); // 45° → borné à 40° (au-delà du centre)
  });
  test("clic simple sur le bras levé, en pause : la lecture reprend", () => {
    expect(armRelease({ endAngle: ARM_REST, movedPx: 1, playing: false, durationMs: 1000 })).toEqual({ kind: "play" });
  });
  test("clic simple pendant la lecture : rien", () => {
    expect(armRelease({ endAngle: 20, movedPx: 2, playing: true, durationMs: 1000 })).toEqual({ kind: "none" });
  });
  test("glisser hors du disque pendant la lecture : pause", () => {
    expect(armRelease({ endAngle: 6, movedPx: 40, playing: true, durationMs: 1000 })).toEqual({ kind: "pause" });
  });
  test("glisser sur le disque : seek, et reprise si en pause", () => {
    expect(armRelease({ endAngle: 26, movedPx: 40, playing: true, durationMs: 1000 })).toEqual({ kind: "seek", positionMs: 500, resume: false });
    expect(armRelease({ endAngle: 26, movedPx: 40, playing: false, durationMs: 1000 })).toEqual({ kind: "seek", positionMs: 500, resume: true });
  });
  test("glisser sur un flux sans durée : pas de seek, juste la lecture si en pause", () => {
    expect(armRelease({ endAngle: 26, movedPx: 40, playing: true, durationMs: 0 })).toEqual({ kind: "none" });
    expect(armRelease({ endAngle: 26, movedPx: 40, playing: false, durationMs: 0 })).toEqual({ kind: "play" });
  });
});
