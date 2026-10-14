export type Route = "platine" | "bibliotheque" | "artiste" | "personnaliser" | "reglages";
const ROUTES: Route[] = ["platine", "bibliotheque", "artiste", "personnaliser", "reglages"];
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
