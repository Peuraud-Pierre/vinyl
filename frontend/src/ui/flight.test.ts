import { expect, test } from "vitest";
import { albumCoverRect, flightFrames } from "./flight";

test("place de la pochette sur la page album : à gauche, 36 % de la largeur, 560 px au plus", () => {
  expect(albumCoverRect(1280)).toEqual({ left: 92, top: 134, size: 460.8 });
  expect(albumCoverRect(2400).size).toBe(560);
});
test("trajectoire : part du boîtier incliné, arrive à plat sur la page", () => {
  const from = { left: 700, top: 200, width: 90, height: 300 };
  const [a, b] = flightFrames(from, albumCoverRect(1280), 72);
  expect(a).toMatchObject({ left: "700px", top: "200px", width: "90px", height: "300px" });
  expect(a.transform).toContain("rotateY(72deg)");
  expect(b).toMatchObject({ left: "92px", top: "134px", width: "460.8px", height: "460.8px", transform: "none" });
});
