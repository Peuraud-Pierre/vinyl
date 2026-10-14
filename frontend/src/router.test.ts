import { expect, test } from "vitest";
import { parseRoute } from "./router";

test("route par défaut", () => { expect(parseRoute("")).toEqual({ route: "platine", param: "" }); });
test("route avec paramètre encodé", () => {
  expect(parseRoute("#/artiste/Beyonc%C3%A9")).toEqual({ route: "artiste", param: "Beyoncé" });
});
test("route inconnue → platine", () => { expect(parseRoute("#/nimporte")).toEqual({ route: "platine", param: "" }); });
test("anciennes routes de bibliothèque → étagère avec l'onglet", () => {
  expect(parseRoute("#/albums")).toEqual({ route: "bibliotheque", param: "albums" });
  expect(parseRoute("#/playlists")).toEqual({ route: "bibliotheque", param: "playlists" });
  expect(parseRoute("#/likes")).toEqual({ route: "bibliotheque", param: "likes" });
  expect(parseRoute("#/artistes")).toEqual({ route: "bibliotheque", param: "artistes" });
});
test("étagère : onglet par défaut et onglet inconnu → albums", () => {
  expect(parseRoute("#/bibliotheque")).toEqual({ route: "bibliotheque", param: "albums" });
  expect(parseRoute("#/bibliotheque/n-importe")).toEqual({ route: "bibliotheque", param: "albums" });
  expect(parseRoute("#/bibliotheque/playlists")).toEqual({ route: "bibliotheque", param: "playlists" });
});
