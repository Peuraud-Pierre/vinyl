import { expect, test } from "vitest";
import { albumHeaders } from "./queuegroups";

const t = (title: string, album: string) => ({ id: title, number: 0, title, artist: "A", album, durationMs: 0, loved: false });

test("l'album n'apparaît qu'en tête de groupe, et pas pour l'album du morceau en cours", () => {
  const rows = albumHeaders("Colors - Single", [t("Heart Flutters", "Colors - Single"), t("Star Hoppers", "Colors - Single"), t("Get Lucky", "Random Access Memories"), t("Lose Yourself", "Random Access Memories")]);
  expect(rows.map((r) => r.header)).toEqual(["", "", "Random Access Memories", ""]);
});
test("si le premier suivant vient d'un autre album, il porte l'en-tête", () => {
  const rows = albumHeaders("Discovery", [t("Heart Flutters", "Colors - Single")]);
  expect(rows[0]!.header).toBe("Colors - Single");
});
test("album inconnu : pas d'en-tête", () => {
  expect(albumHeaders("", [t("X", "")])[0]!.header).toBe("");
});
