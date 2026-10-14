import { expect, test } from "vitest";
import { nextRepeat } from "./repeat";

test("répéter : aucun → tout → un seul → aucun", () => {
  expect(nextRepeat("off")).toBe("all");
  expect(nextRepeat("all")).toBe("one");
  expect(nextRepeat("one")).toBe("off");
  expect(nextRepeat("??")).toBe("all");
});

test("Spotify ne connaît pas « un seul » : aucun ↔ tout", () => {
  expect(nextRepeat("off", "spotify")).toBe("all");
  expect(nextRepeat("all", "spotify")).toBe("off");
  expect(nextRepeat("all", "applemusic")).toBe("one");
});
