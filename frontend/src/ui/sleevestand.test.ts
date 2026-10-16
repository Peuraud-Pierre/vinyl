import { expect, test } from "vitest";
import { mountSleeveStand } from "./sleevestand";

test("support de pochette : socle en bois, rainure et « EN ÉCOUTE » gravé", () => {
  const el = document.createElement("div");
  mountSleeveStand(el);
  expect(el.querySelector(".vy-stand-groove")).not.toBeNull();
  expect(el.querySelector(".vy-stand-front")!.textContent).toBe("EN ÉCOUTE");
  expect(el.getAttribute("aria-hidden")).toBe("true");
});
