import { expect, test } from "vitest";
import { discBackground } from "./textures";

const colors: [string, string] = ["#ffb300", "#fff3d6"];

test("noir", () => { expect(discBackground("black", colors, "")).toContain("#111"); });
test("picture disc utilise la pochette", () => {
  expect(discBackground("picture", colors, "/artwork/a.img")).toContain('url("/artwork/a.img")');
});
test("picture disc sans pochette retombe sur le noir", () => {
  expect(discBackground("picture", colors, "")).toBe(discBackground("black", colors, ""));
});
test("yolk et split utilisent les deux couleurs", () => {
  for (const t of ["yolk", "split"] as const) {
    const bg = discBackground(t, colors, "");
    expect(bg).toContain("#ffb300");
    expect(bg).toContain("#fff3d6");
  }
});
test("une URL avec guillemets est échappée", () => {
  expect(discBackground("picture", colors, 'x"y')).toContain('url("x\\"y")');
});
