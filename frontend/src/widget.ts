import "./styles.css";
import { api, applyAccent, onSettings, onSnapshot, playPause, type Settings, type Snapshot } from "./api";
import { Turntable } from "./turntable/turntable";

const tt = new Turntable(document.getElementById("disc")!, { variant: "widget" });
const hint = document.getElementById("hint")!;

function showSnapshot(s: Snapshot) {
  tt.update(s);
  const msg = s.problem === "no-player" ? "Lance Musique<br>ou Spotify" : s.problem === "permission" ? "Autorisation<br>requise" : "";
  hint.innerHTML = msg;
  hint.classList.toggle("hidden", !msg);
  hint.classList.toggle("flex", !!msg);
}

function showSettings(s: Settings) {
  tt.setAppearance(s);
  applyAccent(s.accent);
}

document.getElementById("prev")!.onclick = () => api.Previous();
document.getElementById("pp")!.onclick = () => void playPause().catch(() => {});
document.getElementById("next")!.onclick = () => api.Next();
document.getElementById("open")!.onclick = () => api.OpenMainWindow();

onSnapshot(showSnapshot);
onSettings(showSettings);
api.GetSettings().then(showSettings);
api.GetSnapshot().then(showSnapshot);
