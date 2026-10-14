import { describe, expect, test } from "vitest";
import { averageColor, hexToRgb, luminance, paletteFor, rgbToHex } from "./theme";

describe("couleurs", () => {
  test("hex ↔ rgb", () => {
    expect(hexToRgb("#8a5f57")).toEqual([138, 95, 87]);
    expect(rgbToHex([138, 95, 87])).toBe("#8a5f57");
    expect(hexToRgb("pas une couleur")).toBeNull();
  });

  test("luminance : noir 0, blanc 1", () => {
    expect(luminance([0, 0, 0])).toBe(0);
    expect(luminance([255, 255, 255])).toBeCloseTo(1, 5);
  });

  test("couleur moyenne : les pixels saturés comptent plus que les gris", () => {
    // 3 pixels gris moyens et 1 pixel orange vif (RGBA).
    const px = new Uint8ClampedArray([128, 128, 128, 255, 128, 128, 128, 255, 128, 128, 128, 255, 240, 120, 30, 255]);
    const [r, , b] = averageColor(px)!;
    expect(r).toBeGreaterThan(b + 40);
  });

  test("couleur moyenne : pixels transparents ignorés, image vide → null", () => {
    expect(averageColor(new Uint8ClampedArray([255, 0, 0, 0]))).toBeNull();
    expect(averageColor(new Uint8ClampedArray([]))).toBeNull();
  });
});

describe("palette", () => {
  test("pochette sombre : fond sombre, texte clair", () => {
    const p = paletteFor("#3a2622");
    expect(p.light).toBe(false);
    expect(p.ink).toBe("#fbf3ef");
  });

  test("pochette claire : texte sombre (page In Utero)", () => {
    const p = paletteFor("#e9e0a6");
    expect(p.light).toBe(true);
    expect(luminance(hexToRgb(p.ink)!)).toBeLessThan(0.1);
  });

  test("couleur invalide : palette de repli", () => {
    expect(paletteFor("n'importe").tint).toBe("#8a5f57");
  });

  test("le fond n'est jamais ni noir pur ni blanc pur", () => {
    for (const c of ["#000000", "#ffffff", "#ff0000"]) {
      const l = luminance(hexToRgb(paletteFor(c).tint)!);
      expect(l).toBeGreaterThan(0.02);
      expect(l).toBeLessThan(0.8);
    }
  });
});
