import "./styles.css";
import { api, applyAccent, onSettings, onWidgetUpdate, type Settings } from "./api";
import { render as renderWidgetUpdate, type WidgetStatus } from "./widgetupdate";
import { parseRoute, type Route } from "./router";
import * as platine from "./views/platine";
import * as personnaliser from "./views/personnaliser";
import * as reglages from "./views/reglages";
import * as bibliotheque from "./views/bibliotheque";
import * as artiste from "./views/artiste";

type View = { mount: (el: HTMLElement, param: string) => () => void };
const lib: View = { mount: (el) => bibliotheque.mount(el, parseRoute(location.hash).route) };
const views: Partial<Record<Route, View>> = {
  platine, personnaliser, reglages, artiste,
  playlists: lib, albums: lib, likes: lib, artistes: lib,
};

const viewEl = document.getElementById("view")!;
let cleanup: () => void = () => {};

function route() {
  const { route, param } = parseRoute(location.hash);
  cleanup();
  viewEl.innerHTML = "";
  cleanup = (views[route] ?? platine).mount(viewEl, param);
  document.querySelectorAll<HTMLElement>("[data-route]").forEach((a) => a.classList.toggle("active", a.dataset.route === route));
}

window.addEventListener("hashchange", route);
route();

const source = document.getElementById("source") as HTMLSelectElement;
const syncSettings = (s: Settings) => {
  applyAccent(s.accent);
  source.value = s.source;
};
api.GetSettings().then(syncSettings);
onSettings(syncSettings);
source.onchange = () => api.UpdateSettings({ source: source.value });

window.addEventListener("keydown", (e) => {
  if ((e.target as HTMLElement).closest("input, select, textarea")) return;
  if (e.code === "Space") {
    e.preventDefault();
    api.PlayPause();
  }
  if (e.key === "ArrowLeft") api.Previous();
  if (e.key === "ArrowRight") api.Next();
});

const spStatus = document.getElementById("spotify-status")!;
const refreshSpotify = async () => {
  spStatus.textContent = (await api.SpotifyStatus()) ? "Spotify : connecté" : "Spotify : non connecté";
};
refreshSpotify();
window.addEventListener("hashchange", refreshSpotify);

const widgetUpdateEl = document.getElementById("widget-update")!;
const showWidgetUpdate = (s: WidgetStatus) =>
  renderWidgetUpdate(widgetUpdateEl, s, { update: () => void api.UpdateWidget().catch(console.error) });
api.WidgetUpdateStatus().then(showWidgetUpdate);
onWidgetUpdate(showWidgetUpdate);
