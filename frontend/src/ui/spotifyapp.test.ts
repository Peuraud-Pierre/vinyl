import { expect, test, vi } from "vitest";
import { isClientID, mountSpotifyAppWizard, REDIRECT_URI } from "./spotifyapp";

const ID = "0123456789abcdef0123456789abcdef";
const deps = (over = {}) => ({
  clientId: "",
  openURL: vi.fn(),
  copy: vi.fn(() => Promise.resolve()),
  save: vi.fn((_: string) => Promise.resolve()),
  connect: vi.fn(() => Promise.resolve()),
  verify: vi.fn(() => Promise.resolve()),
  ...over,
});
const flush = () => new Promise((r) => setTimeout(r, 0));

test("Client ID : 32 caractères hexadécimaux, espaces autour tolérés", () => {
  expect(isClientID(ID)).toBe(true);
  expect(isClientID(`  ${ID}\n`)).toBe(true);
  expect(isClientID("abc")).toBe(false);
  expect(isClientID(ID.slice(0, 31) + "z")).toBe(false);
});

test("étape 1 : ouvre la création d'app et copie l'adresse de retour", () => {
  const el = document.createElement("div");
  const d = deps();
  mountSpotifyAppWizard(el, d);
  expect(el.textContent).toContain(REDIRECT_URI);
  el.querySelector<HTMLButtonElement>("[data-open]")!.click();
  expect(d.openURL).toHaveBeenCalledWith("https://developer.spotify.com/dashboard/create");
  el.querySelector<HTMLButtonElement>("[data-copy]")!.click();
  expect(d.copy).toHaveBeenCalledWith(REDIRECT_URI);
});

test("Client ID invalide : message, rien d'enregistré", async () => {
  const el = document.createElement("div");
  const d = deps();
  mountSpotifyAppWizard(el, d);
  el.querySelector<HTMLInputElement>("[data-cid]")!.value = "pas un id";
  el.querySelector<HTMLButtonElement>("[data-go]")!.click();
  await flush();
  expect(d.save).not.toHaveBeenCalled();
  expect(el.querySelector("[data-msg]")!.textContent).toContain("32 caractères");
});

test("Client ID valide : enregistré, connexion, vérification, succès", async () => {
  const el = document.createElement("div");
  const d = deps();
  mountSpotifyAppWizard(el, d);
  el.querySelector<HTMLInputElement>("[data-cid]")!.value = ` ${ID} `;
  el.querySelector<HTMLButtonElement>("[data-go]")!.click();
  await flush();
  expect(d.save).toHaveBeenCalledWith(ID);
  expect(d.connect).toHaveBeenCalled();
  expect(d.verify).toHaveBeenCalled();
  expect(el.querySelector("[data-msg]")!.textContent).toContain("C'est bon");
});

test("vérification refusée par Spotify : la raison est affichée", async () => {
  const el = document.createElement("div");
  const d = deps({ verify: vi.fn(() => Promise.reject(new Error("spotify: HTTP 403 : Forbidden"))) });
  mountSpotifyAppWizard(el, d);
  el.querySelector<HTMLInputElement>("[data-cid]")!.value = ID;
  el.querySelector<HTMLButtonElement>("[data-go]")!.click();
  await flush();
  expect(el.querySelector("[data-msg]")!.textContent).toContain("HTTP 403 : Forbidden");
});

test("app perso déjà utilisée : retour possible à l'app Vinyl", async () => {
  const el = document.createElement("div");
  const d = deps({ clientId: ID });
  mountSpotifyAppWizard(el, d);
  expect(el.querySelector<HTMLInputElement>("[data-cid]")!.value).toBe(ID);
  el.querySelector<HTMLButtonElement>("[data-reset]")!.click();
  await flush();
  expect(d.save).toHaveBeenCalledWith("");
});
