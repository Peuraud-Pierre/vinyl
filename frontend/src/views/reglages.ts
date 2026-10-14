import { api, type Settings } from "../api";
import { go } from "../router";
import { icons } from "../ui/icons";

const escapeAttr = (s: string) => s.replace(/[&"<>]/g, (c) => ({ "&": "&amp;", '"': "&quot;", "<": "&lt;", ">": "&gt;" })[c]!);

export function mount(el: HTMLElement): () => void {
  el.innerHTML = `
    <div class="flex min-h-screen items-center justify-center py-16" style="padding-right:130px">
      <div class="panel flex w-[620px] flex-col" role="dialog" aria-label="Réglages">
        <div class="flex items-center justify-between pb-4">
          <h1 class="text-[30px] font-extrabold">Réglages</h1>
          <button id="close" class="icon-btn h-11 w-11 rounded-full" style="background:rgba(255,255,255,.55)" aria-label="Fermer">${icons.close}</button>
        </div>
        <section class="flex flex-col gap-3 border-t py-5" style="border-color:rgba(42,29,27,.14)">
          <h2>Source</h2>
          <div id="sources" role="radiogroup" aria-label="Source" class="flex gap-1 self-start rounded-full p-1" style="background:rgba(255,255,255,.4)"></div>
        </section>
        <section id="spotify-section" class="flex flex-col gap-3 border-t py-5" style="border-color:rgba(42,29,27,.14)"></section>
        <section class="flex flex-col gap-3 border-t py-5" style="border-color:rgba(42,29,27,.14)">
          <h2>Son</h2>
          <label class="flex cursor-pointer items-center justify-between text-[17px]">Craquement quand l'aiguille se pose
            <input type="checkbox" id="crackle" class="h-5 w-5" style="accent-color:#2a1d1b"></label>
        </section>
      </div>
    </div>`;
  el.querySelector<HTMLElement>("#close")!.onclick = () => go("platine");
  const SOURCES: [string, string][] = [["auto", "Auto"], ["applemusic", "Apple Music"], ["spotify", "Spotify"]];
  const renderSources = () => {
    const box = el.querySelector<HTMLElement>("#sources")!;
    box.innerHTML = SOURCES.map(([id, label]) => `<button role="radio" data-src="${id}" aria-checked="${cur?.source === id}" class="pill h-[42px] cursor-pointer text-[16px] ${cur?.source === id ? "pill-on" : ""}">${label}</button>`).join("");
    box.querySelectorAll<HTMLElement>("[data-src]").forEach((b) => (b.onclick = async () => {
      cur = await api.UpdateSettings({ source: b.dataset.src! });
      renderSources();
    }));
  };
  const crackle = el.querySelector<HTMLInputElement>("#crackle")!;
  const section = el.querySelector<HTMLElement>("#spotify-section")!;
  let cur: Settings | null = null;

  const renderSpotify = async () => {
    if (!cur) return;
    const connected = await api.SpotifyStatus();
    section.innerHTML = `
      <h2>Spotify</h2>
      <p class="text-[14px] leading-relaxed" style="color:rgba(42,29,27,.8)">Crée une app gratuite sur developer.spotify.com/dashboard, ajoute l'URI de redirection
        <code class="rounded-md px-1.5" style="background:rgba(255,255,255,.55)">http://127.0.0.1:43821/callback</code>, puis colle son Client ID ici.</p>
      <div class="flex gap-2.5">
        <label for="cid" class="sr-only">Client ID Spotify</label>
        <input id="cid" class="field flex-1" placeholder="Client ID" value="${escapeAttr(cur.spotifyClientId)}">
        <button id="sp-btn" class="btn-dark cursor-pointer">${connected ? "Se déconnecter" : "Se connecter"}</button>
      </div>
      <p id="sp-msg" class="text-[14px]" style="color:rgba(42,29,27,.8)">${connected ? "Connecté." : ""}</p>`;
    const cid = section.querySelector<HTMLInputElement>("#cid")!;
    const msg = section.querySelector<HTMLElement>("#sp-msg")!;
    const saveClientID = async () => {
      cur = await api.UpdateSettings({ spotifyClientId: cid.value.trim() });
    };
    cid.onchange = saveClientID;
    const btn = section.querySelector<HTMLButtonElement>("#sp-btn")!;
    btn.onclick = async () => {
      btn.disabled = true;
      try {
        if (connected) await api.SpotifyLogout();
        else {
          await saveClientID(); // le Client ID tapé juste avant doit être enregistré avant la connexion
          msg.textContent = "Termine la connexion dans ton navigateur…";
          await api.SpotifyLogin();
        }
        renderSpotify();
      } catch (e) {
        msg.textContent = String(e);
        btn.disabled = false;
      }
    };
  };

  api.GetSettings().then((s) => {
    cur = s;
    crackle.checked = s.crackle;
    renderSources();
    renderSpotify();
  });
  crackle.onchange = async () => {
    cur = await api.UpdateSettings({ crackle: crackle.checked });
  };
  return () => {};
}
