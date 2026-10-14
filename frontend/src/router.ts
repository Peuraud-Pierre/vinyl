export type Route = "platine" | "bibliotheque" | "artiste" | "album" | "personnaliser" | "reglages";
const ROUTES: Route[] = ["platine", "bibliotheque", "artiste", "album", "personnaliser", "reglages"];
export const TABS = ["albums", "playlists", "artistes", "likes"] as const;
export type Tab = (typeof TABS)[number];

export function parseRoute(hash: string): { route: Route; param: string } {
  const [, name = "", ...rest] = hash.replace(/^#/, "").split("/");
  // Anciennes routes de la barre latérale : un onglet de l'étagère.
  if ((TABS as readonly string[]).includes(name)) return { route: "bibliotheque", param: name };
  const route = (ROUTES as string[]).includes(name) ? (name as Route) : "platine";
  let param = route === "platine" ? "" : decodeURIComponent(rest.join("/"));
  if (route === "bibliotheque" && !(TABS as readonly string[]).includes(param)) param = "albums";
  return { route, param };
}

export function go(route: Route, param = ""): void {
  location.hash = `#/${route}${param ? "/" + encodeURIComponent(param) : ""}`;
}

// Paramètre de la page album : « source/id » (l'ID peut contenir / ou des séparateurs).
export const albumParam = (source: string, id: string) => `${source}/${id}`;
export function parseAlbumParam(param: string): { source: string; id: string } {
  const i = param.indexOf("/");
  return i < 0 ? { source: "applemusic", id: param } : { source: param.slice(0, i), id: param.slice(i + 1) };
}
