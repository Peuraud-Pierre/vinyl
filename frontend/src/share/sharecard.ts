import { api } from "../api";
import { renderStory } from "./story";
import { imageTint } from "../ui/theme";

export interface ShareInput { title: string; subtitle: string; artworkUrl: string; }

export async function openShareCard(input: ShareInput): Promise<void> {
  const s = await api.GetSettings();
  const tint = (await imageTint(input.artworkUrl)) ?? s.accent;
  const canvas = await renderStory(input, s, tint);
  const overlay = document.createElement("div");
  overlay.className = "fixed inset-0 z-50 flex items-center justify-center bg-black/70";
  overlay.innerHTML = `<div class="flex flex-col items-center gap-5 rounded-[34px] bg-neutral-900/90 p-6">
      <img class="h-[72vh] rounded-[28px] shadow-2xl" alt="Aperçu du partage">
      <p id="share-msg" class="text-sm text-neutral-400"></p>
      <div class="flex gap-3">
        <button id="copy" class="rounded-full bg-white px-6 py-2.5 text-[16px] font-semibold text-neutral-900">Copier</button>
        <button id="save" class="rounded-full bg-white/15 px-6 py-2.5 text-[16px] font-semibold text-white">Enregistrer…</button>
        <button id="close" class="rounded-full px-5 py-2.5 text-[16px] text-white/70">Fermer</button>
      </div></div>`;
  document.body.appendChild(overlay);
  const msg = overlay.querySelector<HTMLElement>("#share-msg")!;
  let dataUrl = "";
  try {
    dataUrl = canvas.toDataURL("image/png");
  } catch {
    // Pochette distante sans CORS : le canvas est « sali », on regénère sans elle.
    msg.textContent = "La pochette ne peut pas être exportée (image protégée) : la carte est générée sans elle.";
    dataUrl = (await renderStory({ ...input, artworkUrl: "" }, s, tint)).toDataURL("image/png");
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
