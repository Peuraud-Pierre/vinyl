import "./styles.css";
import { api, applyAccent, onSettings, onWidgetUpdate, playPause } from "./api";
import { parseRoute, type Route } from "./router";
import { mountShell } from "./shell/shell";
import * as platine from "./views/platine";
import * as personnaliser from "./views/personnaliser";
import * as reglages from "./views/reglages";
import * as bibliotheque from "./views/bibliotheque";
import * as artiste from "./views/artiste";
import * as album from "./views/album";
import { render as renderWidgetUpdate, type WidgetStatus } from "./widgetupdate";
import { animateOnce } from "./ui/animate";
import { platform } from "./ui/platform";
import { mountUpdateBanner, throttledCheck, type UpdateInfo } from "./ui/updatebanner";
import { Events } from "@wailsio/runtime";
import { initPhonePreview } from "./ui/phoneframe";

type View = { mount: (el: HTMLElement, param: string) => () => void };
const views: Record<Route, View> = { platine, bibliotheque, artiste, album, personnaliser, reglages };

const viewEl = document.getElementById("view")!;
let cleanup: () => void = () => {};
let accent = "#f5c542";

const shell = mountShell(document.getElementById("shell")!, () => accent);
// Version de développement : l'aperçu téléphone reprend s'il était actif (jamais dans une release).
api.Version().then((v) => { if (v === "dev") initPhonePreview(); }).catch(() => {});

// Changement d'écran : l'ancien s'efface en flou, le nouveau apparaît (comme MD Vinyl).
const LEAVE_MS = 200;
let routing = 0;
async function route() {
  const token = ++routing;
  const { route, param } = parseRoute(location.hash);
  const old = viewEl.firstElementChild;
  if (old) {
    old.classList.add("vy-leave");
    await new Promise((r) => setTimeout(r, LEAVE_MS));
    if (token !== routing) return; // un autre changement est arrivé entre-temps
  }
  cleanup();
  viewEl.innerHTML = "";
  viewEl.scrollTop = 0;
  cleanup = views[route].mount(viewEl, param);
  animateOnce(viewEl.firstElementChild, "vy-enter");
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
    playPause().catch(() => {});
  }
  if (e.key === "ArrowLeft") api.Previous();
  if (e.key === "ArrowRight") api.Next();
});

// Mises à jour de Vinyl : vérification au lancement, puis annonces du backend (toutes les heures) et
// recherche manuelle depuis Réglages (événement DOM « vinyl:update »).
const updateBanner = mountUpdateBanner(document.getElementById("app-update")!, {
  install: () => api.InstallUpdate(),
  openRelease: (v) => void api.ReleaseURL(v).then((u) => api.OpenURL(u)),
});
// Au démarrage, puis à chaque retour sur la fenêtre (au plus toutes les 15 min) : une version publiée
// pendant que Vinyl est ouvert est proposée sans attendre la vérification horaire.
const checkUpdates = throttledCheck(() => api.CheckForUpdate(), (i) => updateBanner.show(i), 15 * 60_000);
checkUpdates();
window.addEventListener("focus", checkUpdates);
document.addEventListener("visibilitychange", () => { if (!document.hidden) checkUpdates(); });
Events.On("appupdate", (e) => updateBanner.show(e.data as UpdateInfo));
window.addEventListener("vinyl:update", (e) => updateBanner.show((e as CustomEvent<UpdateInfo>).detail));
Events.On("wails:updater:download-progress", (e) => {
  const p = e.data as { written: number; total: number }; // updater.Progress
  updateBanner.progress(p.written, p.total);
});
Events.On("wails:updater:error", () => updateBanner.installFailed());

const widgetUpdateEl = document.getElementById("widget-update")!;
const showWidgetUpdate = (s: WidgetStatus) =>
  renderWidgetUpdate(widgetUpdateEl, s, { update: () => void api.UpdateWidget().catch(console.error) });
// Widget de bureau (WidgetKit) : Mac seulement.
platform().then((p) => {
  if (p !== "darwin") return;
  api.WidgetUpdateStatus().then(showWidgetUpdate);
  onWidgetUpdate(showWidgetUpdate);
});
