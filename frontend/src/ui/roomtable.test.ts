import { expect, test } from "vitest";
import { mountRoomTable } from "./roomtable";

// Grande table : plateau, ceinture avec un tiroir (façade + poignée), deux pieds dans l'ombre dessous.
test("la table a un plateau, une ceinture avec un tiroir et deux pieds", () => {
  const el = document.createElement("div");
  const t = mountRoomTable(el);
  expect(el.querySelector(".vy-rt-top")).not.toBeNull();
  expect(el.querySelector(".vy-rt-apron .vy-rt-drawer .vy-rt-knob")).not.toBeNull();
  expect(el.querySelectorAll(".vy-rt-leg").length).toBe(2);
  t.setScale(0.8);
  expect(el.style.getPropertyValue("--k")).toBe("0.8");
});
