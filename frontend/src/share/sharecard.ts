import { api, type Settings } from "../api";
import { CARD_H, CARD_W, fitText } from "./layout";

export interface ShareInput { title: string; subtitle: string; artworkUrl: string; }

function loadImage(src: string): Promise<HTMLImageElement | null> {
  if (!src) return Promise.resolve(null);
  return new Promise((resolve) => {
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.onload = () => resolve(img);
    img.onerror = () => resolve(null);
    img.src = src;
  });
}

export async function renderCard(input: ShareInput, s: Settings): Promise<HTMLCanvasElement> {
  const c = document.createElement("canvas");
  c.width = CARD_W;
  c.height = CARD_H;
  const g = c.getContext("2d")!;
  const bg = g.createRadialGradient(CARD_W / 2, 520, 50, CARD_W / 2, 520, 900);
  bg.addColorStop(0, s.accent + "66");
  bg.addColorStop(1, "#141414");
  g.fillStyle = bg;
  g.fillRect(0, 0, CARD_W, CARD_H);

  const cx = CARD_W / 2, cy = 560, r = 400;
  const art = await loadImage(input.artworkUrl);
  const [c1 = "#ffb300", c2 = "#fff3d6"] = s.discColors;

  // Disque
  g.save();
  g.beginPath(); g.arc(cx, cy, r, 0, Math.PI * 2); g.clip();
  if (s.texture === "picture" && art) g.drawImage(art, cx - r, cy - r, r * 2, r * 2);
  else if (s.texture === "split") { g.fillStyle = c1; g.fillRect(cx - r, cy - r, r, r * 2); g.fillStyle = c2; g.fillRect(cx, cy - r, r, r * 2); }
  else if (s.texture === "yolk") { g.fillStyle = c2; g.fillRect(cx - r, cy - r, r * 2, r * 2); g.fillStyle = c1; g.beginPath(); g.arc(cx, cy, r * 0.42, 0, Math.PI * 2); g.fill(); }
  else if (s.texture === "solid") { g.fillStyle = c1 + "cc"; g.fillRect(cx - r, cy - r, r * 2, r * 2); }
  else { g.fillStyle = "#111"; g.fillRect(cx - r, cy - r, r * 2, r * 2); }
  g.strokeStyle = "rgba(0,0,0,.25)";
  g.lineWidth = 1;
  for (let rr = r * 0.4; rr < r; rr += 4) { g.beginPath(); g.arc(cx, cy, rr, 0, Math.PI * 2); g.stroke(); }
  g.restore();

  // Étiquette
  const lr = r * 0.36;
  g.save();
  g.beginPath(); g.arc(cx, cy, lr, 0, Math.PI * 2); g.clip();
  if (art) g.drawImage(art, cx - lr, cy - lr, lr * 2, lr * 2);
  else { g.fillStyle = s.accent; g.fillRect(cx - lr, cy - lr, lr * 2, lr * 2); }
  g.restore();
  g.fillStyle = "#111";
  g.beginPath(); g.arc(cx, cy, 10, 0, Math.PI * 2); g.fill();

  // Textes
  g.textAlign = "center";
  g.fillStyle = "#fff";
  g.font = "700 64px -apple-system, sans-serif";
  g.fillText(fitText((t) => g.measureText(t).width, input.title, CARD_W - 160), cx, 1100);
  g.fillStyle = "#bbb";
  g.font = "400 40px -apple-system, sans-serif";
  g.fillText(fitText((t) => g.measureText(t).width, input.subtitle, CARD_W - 160), cx, 1170);
  return c;
}

export async function openShareCard(input: ShareInput): Promise<void> {
  const s = await api.GetSettings();
  const canvas = await renderCard(input, s);
  const overlay = document.createElement("div");
  overlay.className = "fixed inset-0 z-50 flex items-center justify-center bg-black/70";
  overlay.innerHTML = `<div class="flex flex-col items-center gap-4 rounded-xl bg-neutral-900 p-6">
      <img class="h-[60vh] rounded-lg" alt="Carte de partage">
      <p id="share-msg" class="text-sm text-neutral-400"></p>
      <div class="flex gap-3">
        <button id="copy" class="rounded px-4 py-1.5 text-neutral-900" style="background:var(--accent)">Copier</button>
        <button id="save" class="rounded border border-white/30 px-4 py-1.5">Enregistrer…</button>
        <button id="close" class="rounded px-4 py-1.5 text-neutral-400">Fermer</button>
      </div></div>`;
  document.body.appendChild(overlay);
  const msg = overlay.querySelector<HTMLElement>("#share-msg")!;
  let dataUrl = "";
  try {
    dataUrl = canvas.toDataURL("image/png");
  } catch {
    // Pochette distante sans CORS : le canvas est « sali », on regénère sans elle.
    msg.textContent = "La pochette ne peut pas être exportée (image protégée) : la carte est générée sans elle.";
    dataUrl = (await renderCard({ ...input, artworkUrl: "" }, s)).toDataURL("image/png");
  }
  overlay.querySelector("img")!.src = dataUrl;

  overlay.querySelector("#close")!.addEventListener("click", () => overlay.remove());
  overlay.addEventListener("click", (e) => {
    if (e.target === overlay) overlay.remove();
  });
  overlay.querySelector("#copy")!.addEventListener("click", async () => {
    try {
      const blob = await (await fetch(dataUrl)).blob();
      await navigator.clipboard.write([new ClipboardItem({ "image/png": blob })]);
      msg.textContent = "Copiée dans le presse-papier.";
    } catch {
      msg.textContent = "Copie impossible ici, utilise « Enregistrer… ».";
    }
  });
  overlay.querySelector("#save")!.addEventListener("click", async () => {
    const name = `${input.title || "vinyl"}.png`.replace(/[/:]/g, "-");
    try {
      const path = await api.SaveShareImage(dataUrl.split(",")[1]!, name);
      if (path) msg.textContent = `Enregistrée : ${path}`;
    } catch (e) {
      msg.textContent = String(e);
    }
  });
}
