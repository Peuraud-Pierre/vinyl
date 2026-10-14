import { api, applyAccent, type Settings } from "../api";
import { go } from "../router";
import { STAGE_H, STAGE_W, Turntable } from "../turntable/turntable";
import { icons } from "../ui/icons";

const TEXTURES: [string, string][] = [
  ["black", "Noir fumé"], ["picture", "Picture disc"], ["yolk", "Yolk"], ["split", "Split"], ["solid", "Couleur unie"],
];
const PRESETS = ["#f5c542", "#ff6b6b", "#4ecdc4", "#a78bfa"];

export function mount(el: HTMLElement): () => void {
  const K = 0.42; // aperçu de la scène réduit
  el.innerHTML = `
    <div class="flex min-h-screen items-center justify-center py-16" style="padding-right:130px">
      <div class="panel flex w-[900px] flex-col gap-4" role="dialog" aria-label="Personnaliser">
        <div class="flex items-center justify-between">
          <h1 class="text-[30px] font-extrabold">Personnaliser</h1>
          <button id="close" class="icon-btn h-11 w-11 rounded-full" style="background:rgba(255,255,255,.55)" aria-label="Fermer">${icons.close}</button>
        </div>
        <div class="grid grid-cols-[1fr_300px] items-center gap-6">
          <div class="relative overflow-hidden rounded-3xl" style="height:${STAGE_H * K}px;background:rgba(0,0,0,.18)">
            <div class="absolute" style="left:50%;top:0;width:${STAGE_W}px;transform:translateX(-50%) scale(${K});transform-origin:50% 0"><div id="preview"></div></div>
          </div>
          <div class="flex flex-col gap-5 text-[14px]">
            <section class="flex flex-col gap-2"><h2>Texture du disque</h2><div id="textures" class="flex flex-wrap gap-2"></div></section>
            <section id="colors-section" class="flex flex-col gap-2"><h2>Couleurs du disque</h2>
              <div class="flex gap-3"><input type="color" id="c1" aria-label="Couleur 1"><input type="color" id="c2" aria-label="Couleur 2"></div></section>
            <section class="flex flex-col gap-2"><h2>Couleur de repli</h2>
              <p style="color:rgba(42,29,27,.75)">Utilisée quand le morceau n'a pas de pochette.</p>
              <div id="presets" class="flex items-center gap-2"></div></section>
          </div>
        </div>
      </div>
    </div>`;
  const $ = <T extends HTMLElement>(id: string) => el.querySelector<T>("#" + id)!;
  $("close").onclick = () => go("platine");
  const tt = new Turntable($("preview"), { variant: "stage" });
  tt.update({ problem: "", problemApp: "", nowPlaying: { source: "", trackId: "preview", title: "Aperçu", artist: "", album: "", durationMs: 0, positionMs: 0, state: "playing", artworkUrl: "" } });
  let cur: Settings | null = null;

  const render = () => {
    if (!cur) return;
    $("textures").innerHTML = TEXTURES.map(([id, label]) =>
      `<button data-t="${id}" aria-pressed="${cur!.texture === id}" class="pill cursor-pointer text-[14px] ${cur!.texture === id ? "pill-on" : ""}" style="${cur!.texture === id ? "" : "background:rgba(255,255,255,.35)"}">${label}</button>`).join("");
    $("colors-section").classList.toggle("hidden", cur.texture === "black" || cur.texture === "picture");
    $<HTMLInputElement>("c1").value = cur.discColors[0] ?? "#ffb300";
    $<HTMLInputElement>("c2").value = cur.discColors[1] ?? "#fff3d6";
    $<HTMLInputElement>("c2").classList.toggle("hidden", cur.texture === "solid");
    $("presets").innerHTML = PRESETS.map((c) => `<button data-c="${c}" aria-label="Couleur ${c}" class="h-7 w-7 cursor-pointer rounded-full border-2 ${cur!.accent === c ? "border-[#2a1d1b]" : "border-transparent"}" style="background:${c}"></button>`).join("")
      + `<input type="color" id="accent" value="${cur.accent}" class="h-7 w-8">`;
  };

  const save = async (patch: Partial<Settings>) => {
    if (!cur) return;
    cur = { ...cur, ...patch };
    tt.setAppearance(cur);
    applyAccent(cur.accent);
    render();
    cur = await api.UpdateSettings(patch);
  };

  el.addEventListener("click", (e) => {
    const t = (e.target as HTMLElement).closest<HTMLElement>("[data-t]");
    if (t) save({ texture: t.dataset.t! });
    const c = (e.target as HTMLElement).closest<HTMLElement>("[data-c]");
    if (c) save({ accent: c.dataset.c! });
  });
  el.addEventListener("change", (e) => {
    if (!cur) return;
    const input = e.target as HTMLInputElement;
    if (input.id === "c1") save({ discColors: [input.value, cur.discColors[1] ?? input.value] });
    if (input.id === "c2") save({ discColors: [cur.discColors[0] ?? input.value, input.value] });
    if (input.id === "accent") save({ accent: input.value });
  });

  api.GetSettings().then((s) => {
    cur = s;
    tt.setAppearance(s);
    render();
  });
  return () => tt.destroy();
}
