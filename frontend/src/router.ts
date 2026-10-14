export type Route = "platine" | "playlists" | "albums" | "likes" | "artistes" | "artiste" | "personnaliser" | "reglages";
const ROUTES: Route[] = ["platine", "playlists", "albums", "likes", "artistes", "artiste", "personnaliser", "reglages"];

export function parseRoute(hash: string): { route: Route; param: string } {
  const [, name = "", ...rest] = hash.replace(/^#/, "").split("/");
  const route = (ROUTES as string[]).includes(name) ? (name as Route) : "platine";
  const param = route === "platine" ? "" : decodeURIComponent(rest.join("/"));
  return { route, param };
}

export function go(route: Route, param = ""): void {
  location.hash = `#/${route}${param ? "/" + encodeURIComponent(param) : ""}`;
}
