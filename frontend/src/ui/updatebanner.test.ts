import { expect, test, vi } from "vitest";
import { mountUpdateBanner } from "./updatebanner";

const deps = () => ({ install: vi.fn(() => Promise.resolve(true)), openRelease: vi.fn() });
const info = (version: string) => ({ available: true, version, notes: "", dev: false });

test("nouvelle version : bandeau et bouton Mettre à jour", () => {
  const el = document.createElement("div");
  const d = deps();
  mountUpdateBanner(el, d).show(info("1.0.3"));
  expect(el.hidden).toBe(false);
  expect(el.textContent).toContain("Vinyl 1.0.3 est disponible");
  el.querySelector<HTMLButtonElement>("[data-update]")!.click();
  expect(d.install).toHaveBeenCalledTimes(1);
});

test("rien de nouveau : pas de bandeau", () => {
  const el = document.createElement("div");
  mountUpdateBanner(el, deps()).show({ available: false, version: "", notes: "", dev: false });
  expect(el.hidden).toBe(true);
});

test("téléchargement : pourcentage ; échec : lien vers la release", () => {
  const el = document.createElement("div");
  const d = deps();
  const b = mountUpdateBanner(el, d);
  b.show(info("1.0.3"));
  el.querySelector<HTMLButtonElement>("[data-update]")!.click();
  b.progress(42, 100);
  expect(el.textContent).toContain("42 %");
  b.show(info("1.0.4")); // annonce pendant l'installation : ignorée
  expect(el.textContent).toContain("42 %");
  b.installFailed();
  expect(el.textContent).toContain("La mise à jour a échoué");
  el.querySelector<HTMLButtonElement>("[data-release]")!.click();
  expect(d.openRelease).toHaveBeenCalledWith("1.0.3");
});

test("échec renvoyé par l'installation : lien vers la release", async () => {
  const el = document.createElement("div");
  const b = mountUpdateBanner(el, { install: () => Promise.reject(new Error("x")), openRelease: vi.fn() });
  b.show(info("1.0.3"));
  el.querySelector<HTMLButtonElement>("[data-update]")!.click();
  await Promise.resolve();
  await Promise.resolve();
  expect(el.textContent).toContain("La mise à jour a échoué");
});

// Wails émet aussi « error » quand une vérification échoue (pas de réseau) : rien ne s'affiche.
test("erreur hors installation : pas de bandeau", () => {
  const el = document.createElement("div");
  const b = mountUpdateBanner(el, deps());
  b.installFailed();
  b.progress(10, 100);
  expect(el.hidden).toBe(true);
});

// Version retirée entre-temps (code contributeur retiré) : le bandeau disparaît.
test("plus rien à installer : bandeau masqué", async () => {
  const el = document.createElement("div");
  const b = mountUpdateBanner(el, { install: () => Promise.resolve(false), openRelease: vi.fn() });
  b.show(info("1.1.0-beta.1"));
  el.querySelector<HTMLButtonElement>("[data-update]")!.click();
  await Promise.resolve();
  await Promise.resolve();
  expect(el.hidden).toBe(true);
  b.show(info("1.0.4")); // nouvelle annonce acceptée ensuite
  expect(el.textContent).toContain("Vinyl 1.0.4 est disponible");
});

test("à jour après une annonce : bandeau masqué", () => {
  const el = document.createElement("div");
  const b = mountUpdateBanner(el, deps());
  b.show(info("1.1.0-beta.1"));
  b.show({ available: false, version: "", notes: "", dev: false });
  expect(el.hidden).toBe(true);
});

// Vinyl lancé depuis Téléchargements (macOS le lance en lecture seule) : dire quoi faire.
test("échec expliqué par Vinyl : la raison est affichée", async () => {
  const el = document.createElement("div");
  const reason = "Vinyl ne peut pas se mettre à jour depuis cet emplacement : glisse-le dans le dossier Applications, puis relance-le.";
  const b = mountUpdateBanner(el, { install: () => Promise.reject(new Error(reason)), openRelease: vi.fn() });
  b.show(info("1.0.5"));
  el.querySelector<HTMLButtonElement>("[data-update]")!.click();
  await new Promise((r) => setTimeout(r, 0));
  expect(el.textContent).toContain("glisse-le dans le dossier Applications");
});

test("échec technique : pas de jargon, seulement le lien", async () => {
  const el = document.createElement("div");
  const b = mountUpdateBanner(el, { install: () => Promise.reject(new Error("updater: download: EOF")), openRelease: vi.fn() });
  b.show(info("1.0.5"));
  el.querySelector<HTMLButtonElement>("[data-update]")!.click();
  await new Promise((r) => setTimeout(r, 0));
  expect(el.textContent).not.toContain("EOF");
  expect(el.textContent).toContain("Voir la release");
});
