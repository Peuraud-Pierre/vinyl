import { expect, test } from "vitest";
import { filterItems, matchingTracks, PILE_SIZE, PILE_STEP, pileCurrent, pilePad, settleWithin, sortTracks, spineColors, unavailableLibraryMessage } from "./shelf";

const items = [
  { id: "1", kind: "album", title: "Lumen", subtitle: "Aurore Vale", artworkUrl: "", url: "" },
  { id: "2", kind: "album", title: "Pavés", subtitle: "Gambi Sud", artworkUrl: "", url: "" },
  { id: "3", kind: "album", title: "Écho", subtitle: "Sable", artworkUrl: "", url: "" },
];

test("recherche : titre ou artiste, sans tenir compte des accents ni de la casse", () => {
  expect(filterItems(items, "").map((i) => i.id)).toEqual(["1", "2", "3"]);
  expect(filterItems(items, "aurore").map((i) => i.id)).toEqual(["1"]);
  expect(filterItems(items, "PAVES").map((i) => i.id)).toEqual(["2"]);
  expect(filterItems(items, "echo").map((i) => i.id)).toEqual(["3"]);
  expect(filterItems(items, "zzz")).toEqual([]);
});

test("recherche dans une playlist : titre, artiste ou album ; renvoie les positions d'origine", () => {
  const tracks = [
    { id: "a", number: 1, title: "Lumière", artist: "Aurore Vale", album: "Lumen", durationMs: 0, loved: false },
    { id: "b", number: 2, title: "Pavés", artist: "Gambi Sud", album: "Rue", durationMs: 0, loved: false },
    { id: "c", number: 3, title: "Nuit", artist: "Sable", album: "Écho", durationMs: 0, loved: false },
  ];
  expect(matchingTracks(tracks, " ")).toEqual([0, 1, 2]);
  expect(matchingTracks(tracks, "lumiere")).toEqual([0]);
  expect(matchingTracks(tracks, "GAMBI")).toEqual([1]);
  expect(matchingTracks(tracks, "echo")).toEqual([2]);
  expect(matchingTracks(tracks, "zzz")).toEqual([]);
});

test("couleur de tranche stable pour un même titre, encre lisible", () => {
  const a = spineColors("Lumen");
  expect(spineColors("Lumen")).toEqual(a);
  expect(a.bg).toMatch(/^#[0-9a-f]{6}$/);
  expect(["#1b1513", "#f6efe9"]).toContain(a.ink);
});

test("couleur de tranche depuis la couleur de la pochette", () => {
  expect(spineColors("x", "#efe6c4").ink).toBe("#1b1513");
  expect(spineColors("x", "#101010").ink).toBe("#f6efe9");
});

test("boîtier prêt (pochette + teinte) : on attend, mais jamais plus que le délai", async () => {
  expect(await settleWithin(Promise.resolve("prêt"), 1000)).toBe(true);
  expect(await settleWithin(Promise.reject(new Error("x")), 1000)).toBe(true); // échec : on affiche quand même
  const t0 = Date.now();
  expect(await settleWithin(new Promise(() => {}), 50)).toBe(false); // jamais prêt : affiché après le délai
  expect(Date.now() - t0).toBeLessThan(500);
});

// Vue pile (demande utilisateur) : la pochette au milieu de l'écran sort à droite ; le premier et le
// dernier élément peuvent venir au milieu.
test("vue pile : marge pour amener le premier et le dernier au milieu", () => {
  expect(pilePad(800)).toBe(400 - PILE_SIZE / 2);
  expect(pilePad(200)).toBe(0); // fenêtre minuscule : pas de marge négative
});
test("vue pile : l'élément courant est celui dont la pochette est au milieu de l'écran", () => {
  const h = 800, pad = pilePad(h);
  expect(pileCurrent(0, h, pad, 50)).toBe(0); // au départ : le premier
  expect(pileCurrent(PILE_STEP * 3, h, pad, 50)).toBe(3);
  expect(pileCurrent(PILE_STEP * 3 + PILE_STEP * 0.4, h, pad, 50)).toBe(3); // arrondi au plus proche
  expect(pileCurrent(PILE_STEP * 3 + PILE_STEP * 0.6, h, pad, 50)).toBe(4);
  expect(pileCurrent(1e6, h, pad, 50)).toBe(49); // jamais au-delà du dernier
  expect(pileCurrent(0, h, pad, 0)).toBe(-1); // liste vide
});
test("vue pile : tout en bas, le dernier élément est au milieu", () => {
  const h = 800, pad = pilePad(h), n = 30;
  // Hauteur totale : marges + pile (chaque élément ne dépasse que de PILE_STEP, sauf le dernier, entier).
  const scrollHeight = pad + (n - 1) * PILE_STEP + PILE_SIZE + pad;
  expect(pileCurrent(scrollHeight - h, h, pad, n)).toBe(n - 1);
});

test("Apple Music sans bibliothèque : message selon la plateforme", () => {
  expect(unavailableLibraryMessage("windows")).toBe("La bibliothèque Apple Music n'est disponible que sur Mac.");
  expect(unavailableLibraryMessage("darwin")).toBe("Non disponible pour Apple Music.");
});

test("tri des titres : année de sortie ou durée, ordre d'origine à égalité, année inconnue en dernier", () => {
  const t = (id: string, year: number, durationMs: number) => ({ id, number: 0, title: id, artist: "", album: "", durationMs, loved: false, year });
  const tracks = [t("a", 1999, 200), t("b", 0, 300), t("c", 1975, 200), t("d", 2021, 100), t("e", 1999, 400)];
  expect(sortTracks(tracks, "")).toEqual([0, 1, 2, 3, 4]);
  expect(sortTracks(tracks, "oldest")).toEqual([2, 0, 4, 3, 1]);
  expect(sortTracks(tracks, "newest")).toEqual([3, 0, 4, 2, 1]);
  expect(sortTracks(tracks, "longest")).toEqual([4, 1, 0, 2, 3]);
  expect(sortTracks(tracks, "shortest")).toEqual([3, 0, 2, 1, 4]);
});
