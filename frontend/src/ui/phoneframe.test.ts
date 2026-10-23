import { expect, test } from "vitest";
import { PHONE, phoneFrame } from "./phoneframe";

test("fenêtre de la taille d'un téléphone : le cadre est la fenêtre", () => {
  expect(phoneFrame(390, 844)).toEqual({ w: 390, h: 844, framed: false });
});

test("plein écran : un cadre aux proportions d'un iPhone, centré dans la hauteur", () => {
  const f = phoneFrame(1512, 982);
  expect(f.framed).toBe(true);
  expect(f.h).toBe(982 - 48);
  expect(f.w / f.h).toBeCloseTo(PHONE.w / PHONE.h, 2);
});

test("fenêtre basse et étroite : le cadre tient aussi en largeur", () => {
  const f = phoneFrame(500, 2000);
  expect(f.w).toBeLessThanOrEqual(500 - 48);
  expect(f.w / f.h).toBeCloseTo(PHONE.w / PHONE.h, 2);
});
