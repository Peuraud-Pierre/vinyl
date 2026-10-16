import { expect, test } from "vitest";
import { recall, remember, forget } from "./shelfcache";

test("l'étagère retrouve ses éléments et sa sélection par source et onglet", () => {
  remember("applemusic:albums", { items: [{ id: "a", kind: "album", title: "A", subtitle: "", artworkUrl: "", url: "" }], next: 50, position: 7 });
  const got = recall("applemusic:albums");
  expect(got?.position).toBe(7);
  expect(got?.next).toBe(50);
  expect(got?.items.map((i) => i.id)).toEqual(["a"]);
  expect(recall("applemusic:playlists")).toBeUndefined();
});
test("la copie rendue est indépendante", () => {
  const got = recall("applemusic:albums")!;
  got.items.push({ id: "b", kind: "album", title: "B", subtitle: "", artworkUrl: "", url: "" });
  expect(recall("applemusic:albums")!.items.length).toBe(1);
});

test("forget efface la position mémorisée", () => {
  remember("spotify:playlists", { items: [], next: 0, position: 120 });
  forget("spotify:playlists");
  expect(recall("spotify:playlists")).toBeUndefined();
});
