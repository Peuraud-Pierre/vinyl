import "./styles.css";
import { api, applyAccent, onSettings, onWidgetUpdate } from "./api";
import { parseRoute, type Route } from "./router";
import { mountShell } from "./shell/shell";
import * as platine from "./views/platine";
import * as personnaliser from "./views/personnaliser";
import * as reglages from "./views/reglages";
import * as bibliotheque from "./views/bibliotheque";
import * as artiste from "./views/artiste";
import { render as renderWidgetUpdate, type WidgetStatus } from "./widgetupdate";

type View = { mount: (el: HTMLElement, param: string) => () => void };
const views: Record<Route, View> = { platine, bibliotheque, artiste, personnaliser, reglages };

const viewEl = document.getElementById("view")!;
let cleanup: () => void = () => {};
let accent = "#f5c542";

const shell = mountShell(document.getElementById("shell")!, () => accent);

function route() {
  const { route, param } = parseRoute(location.hash);
  cleanup();
  viewEl.innerHTML = "";
  viewEl.scrollTop = 0;
  cleanup = views[route].mount(viewEl, param);
}
window.addEventListener("hashchange", route);
route();

const syncAccent = (s: { accent: string }) => {
  const changed = s.accent !== accent;
  accent = s.accent;
  applyAccent(s.accent);
  if (changed) shell.refreshTheme();
};
api.GetSettings().then(syncAccent);
onSettings(syncAccent);

window.addEventListener("keydown", (e) => {
  if ((e.target as HTMLElement).closest("input, select, textarea")) return;
  if (e.code === "Space") {
    e.preventDefault();
    api.PlayPause();
  }
  if (e.key === "ArrowLeft") api.Previous();
  if (e.key === "ArrowRight") api.Next();
});

const widgetUpdateEl = document.getElementById("widget-update")!;
const showWidgetUpdate = (s: WidgetStatus) =>
  renderWidgetUpdate(widgetUpdateEl, s, { update: () => void api.UpdateWidget().catch(console.error) });
api.WidgetUpdateStatus().then(showWidgetUpdate);
onWidgetUpdate(showWidgetUpdate);
