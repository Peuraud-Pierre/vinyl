import { expect, test } from "vitest";
import { animateOnce } from "./animate";

// L'animation d'arrivée d'un écran garde son état final (fill both) : un filtre resterait sur toute la
// page, et dans WebKit un filtre fait dessiner la 3D à plat (pile de disques vue par la tranche).
test("la classe d'animation est retirée à la fin, pour ne laisser ni filtre ni transformation", () => {
  const el = document.createElement("div");
  const child = document.createElement("span");
  el.appendChild(child);
  animateOnce(el, "vy-enter");
  expect(el.classList.contains("vy-enter")).toBe(true);
  child.dispatchEvent(new Event("animationend", { bubbles: true })); // animation d'un enfant : ignorée
  expect(el.classList.contains("vy-enter")).toBe(true);
  el.dispatchEvent(new Event("animationend"));
  expect(el.classList.contains("vy-enter")).toBe(false);
});

test("sans élément, rien ne se passe", () => {
  expect(() => animateOnce(null, "vy-enter")).not.toThrow();
});
