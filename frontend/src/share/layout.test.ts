import { expect, test } from "vitest";
import { fitText } from "./layout";

const measure = (s: string) => s.length * 10;

test("texte court inchangé", () => { expect(fitText(measure, "Daft Punk", 200)).toBe("Daft Punk"); });
test("texte long tronqué avec …", () => {
  const out = fitText(measure, "Un titre beaucoup trop long pour la carte", 100);
  expect(out.endsWith("…")).toBe(true);
  expect(measure(out)).toBeLessThanOrEqual(100);
});
test("texte vide", () => { expect(fitText(measure, "", 100)).toBe(""); });
