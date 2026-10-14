import { beforeEach, describe, expect, test, vi } from "vitest";
import { render } from "./widgetupdate";

describe("icône de mise à jour du widget", () => {
  let el: HTMLElement;
  beforeEach(() => {
    el = document.createElement("div");
  });

  test("rien à faire : masquée", () => {
    render(el, { state: "none", reason: "", message: "" }, { update: () => {} });
    expect(el.hidden).toBe(true);
    expect(el.querySelector("button")).toBeNull();
  });

  test("mise à jour disponible : nuage, bulle et clic", () => {
    const update = vi.fn();
    render(el, { state: "available", reason: "expired", message: "Signature du widget expirée" }, { update });
    const btn = el.querySelector("button")!;
    expect(el.hidden).toBe(false);
    expect(btn.title).toBe("Signature du widget expirée");
    expect(btn.querySelector("svg")).not.toBeNull();
    btn.click();
    expect(update).toHaveBeenCalledOnce();
  });

  test("en cours : l'icône tourne et ne réagit pas", () => {
    const update = vi.fn();
    render(el, { state: "running", reason: "", message: "Mise à jour du widget…" }, { update });
    const btn = el.querySelector("button")!;
    expect(btn.classList.contains("animate-spin")).toBe(true);
    expect(btn.disabled).toBe(true);
    btn.click();
    expect(update).not.toHaveBeenCalled();
  });

  test("erreur : ⚠️, bulle avec le message et « Réessayer »", () => {
    const update = vi.fn();
    render(el, { state: "error", reason: "", message: "Installe XcodeGen : brew install xcodegen" }, { update });
    const btn = el.querySelector("button")!;
    expect(btn.textContent).toBe("⚠️");
    btn.click();
    const pop = el.querySelector("[data-popover]")!;
    expect(pop.textContent).toContain("Installe XcodeGen : brew install xcodegen");
    pop.querySelector<HTMLButtonElement>("[data-retry]")!.click();
    expect(update).toHaveBeenCalledOnce();
    btn.click();
    expect(el.querySelector("[data-popover]")).toBeNull();
  });

  test("message hostile affiché comme du texte", () => {
    render(el, { state: "error", reason: "", message: "<img src=x onerror=alert(1)>" }, { update: () => {} });
    el.querySelector("button")!.click();
    expect(el.querySelector("img")).toBeNull();
  });
});
