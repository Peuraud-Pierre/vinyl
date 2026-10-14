import { api, applyAccent, type Settings } from "../api";
import { Turntable } from "../turntable/turntable";

const TEXTURES: [string, string][] = [
  ["black", "Noir classique"], ["picture", "Picture disc"], ["yolk", "Yolk"], ["split", "Split"], ["solid", "Couleur unie"],
];
const PRESETS = ["#f5c542", "#ff6b6b", "#4ecdc4", "#a78bfa"];

export function mount(el: HTMLElement): () => void {
  el.innerHTML = `
    <div class="grid grid-cols-[1fr_320px] gap-8 p-10 pt-14">
      <div id="preview" class="flex items-center justify-center"></div>
      <div class="flex flex-col gap-6 text-sm">
        <section><h2 class="mb-2 font-semibold">Texture du disque</h2><div id="textures" class="flex flex-wrap gap-2"></div></section>
        <section id="colors-section"><h2 class="mb-2 font-semibold">Couleurs du disque</h2>
          <div class="flex gap-3"><input type="color" id="c1"><input type="color" id="c2"></div></section>
        <section><h2 class="mb-2 font-semibold">Couleur du thème</h2>
          <div id="presets" class="flex items-center gap-2"></div></section>
      </div>
    </div>`;
  const $ = <T extends HTMLElement>(id: string) => el.querySelector<T>("#" + id)!;
  const tt = new Turntable($("preview"), { variant: "full" });
  tt.update({ problem: "", problemApp: "", nowPlaying: { source: "", trackId: "preview", title: "", artist: "", album: "", durationMs: 0, positionMs: 0, state: "playing", artworkUrl: "" } });
  let cur: Settings | null = null;

  const render = () => {
    if (!cur) return;
    $("textures").innerHTML = TEXTURES.map(([id, label]) =>
      `<button data-t="${id}" class="rounded-full border px-3 py-1 ${cur!.texture === id ? "border-transparent text-neutral-900" : "border-neutral-700"}" style="${cur!.texture === id ? "background:var(--accent)" : ""}">${label}</button>`).join("");
    $("colors-section").classList.toggle("hidden", cur.texture === "black" || cur.texture === "picture");
    $<HTMLInputElement>("c1").value = cur.discColors[0] ?? "#ffb300";
    $<HTMLInputElement>("c2").value = cur.discColors[1] ?? "#fff3d6";
    $<HTMLInputElement>("c2").classList.toggle("hidden", cur.texture === "solid");
    $("presets").innerHTML = PRESETS.map((c) => `<button data-c="${c}" class="h-6 w-6 rounded-full border-2 ${cur!.accent === c ? "border-white" : "border-transparent"}" style="background:${c}"></button>`).join("")
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
