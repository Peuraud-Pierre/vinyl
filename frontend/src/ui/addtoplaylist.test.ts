import { expect, test } from "vitest";
import type { NowPlaying } from "../api";
import { mountAddToPlaylist } from "./addtoplaylist";

const np = (trackId: string) => ({ source: "applemusic", trackId, title: "Titre", artist: "Artiste" }) as NowPlaying;
const flush = () => new Promise((r) => setTimeout(r, 0));
const setup = (add: (n: NowPlaying, id: string) => Promise<boolean>) => {
  const btn = document.createElement("button"), menu = document.createElement("div");
  document.body.append(btn, menu);
  let loads = 0;
  const ui = mountAddToPlaylist(btn, menu, {
    playlists: async () => { loads++; return [{ id: "P1", kind: "playlist", title: "Ma liste", subtitle: "", artworkUrl: "", url: "" }]; },
    add,
  });
  return { btn, menu, ui, loads: () => loads };
};

test("le menu liste les playlists, un clic ajoute le morceau en cours", async () => {
  const added: string[] = [];
  const { btn, menu, ui, loads } = setup(async (n, id) => { added.push(n.trackId + "→" + id); return true; });
  ui.update(np("T1"));
  btn.click();
  await flush();
  const item = menu.querySelector<HTMLButtonElement>("button[data-id=P1]")!;
  expect(item.textContent).toBe("Ma liste");
  item.click();
  await flush();
  expect(added).toEqual(["T1→P1"]);
  expect(menu.textContent).toContain("Ajouté à « Ma liste »");
  btn.click(); btn.click(); // refermé puis rouvert : la liste est gardée
  await flush();
  expect(loads()).toBe(1);
});

test("déjà dans la playlist : message, rien d'autre", async () => {
  const { btn, menu, ui } = setup(async () => false);
  ui.update(np("T1"));
  btn.click();
  await flush();
  menu.querySelector<HTMLButtonElement>("button[data-id=P1]")!.click();
  await flush();
  expect(menu.textContent).toContain("Déjà dans « Ma liste »");
});

test("changement de morceau : le menu se ferme ; rien en lecture : bouton caché", async () => {
  const { btn, menu, ui } = setup(async () => true);
  ui.update(np("T1"));
  btn.click();
  await flush();
  expect(menu.hidden).toBe(false);
  ui.update(np("T2"));
  expect(menu.hidden).toBe(true);
  ui.update(null);
  expect(btn.hidden).toBe(true);
});
