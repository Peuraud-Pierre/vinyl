import { expect, test } from "vitest";
import { nextRepeat } from "./repeat";

test("répéter : aucun → tout → un seul → aucun", () => {
  expect(nextRepeat("off")).toBe("all");
  expect(nextRepeat("all")).toBe("one");
  expect(nextRepeat("one")).toBe("off");
  expect(nextRepeat("??")).toBe("all");
});
