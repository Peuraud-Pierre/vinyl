import { api, type Settings } from "../api";
import { go } from "../router";
import { icons } from "../ui/icons";
import { mountSpotifyAppWizard } from "../ui/spotifyapp";
import { Clipboard } from "@wailsio/runtime";

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
        <section id="version-section" class="flex flex-col gap-3 border-t py-5" style="border-color:rgba(42,29,27,.14)"></section>
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

  // Version et mises à jour ; section « Contributeur » : code d'accès aux bêtas.
  const vSection = el.querySelector<HTMLElement>("#version-section")!;
  // Le bandeau (main.ts) suit le résultat : nouvelle version affichée, ou masqué si rien de nouveau.
  const announce = (detail: unknown) => window.dispatchEvent(new CustomEvent("vinyl:update", { detail }));
  const renderVersion = async () => {
    const [v, who] = await Promise.all([api.Version(), api.BetaAccess()]);
    vSection.innerHTML = `
      <h2>Version</h2>
      <div class="flex items-center gap-3 text-[17px]">
        <span>${v === "dev" ? "Vinyl (développement)" : `Vinyl ${escapeAttr(v)}`}</span>
        ${v === "dev" ? "" : `<button id="up-check" class="btn-dark cursor-pointer">Rechercher une mise à jour</button>`}
      </div>
      <p id="up-msg" class="text-[14px]"></p>
      <details class="text-[15px]" ${who ? "open" : ""}>
        <summary class="cursor-pointer">Contributeur</summary>
        ${who
          ? `<p class="mt-2">Bêtas activées pour ${escapeAttr(who)}. <button id="beta-off" class="cursor-pointer underline">Retirer</button></p>`
          : `<div class="mt-2 flex gap-2"><input id="beta-code" class="field flex-1" placeholder="Code contributeur" autocomplete="off" spellcheck="false"><button id="beta-on" class="btn-dark cursor-pointer">Activer</button></div><p id="beta-msg" class="mt-1"></p>`}
      </details>`;
    vSection.querySelector<HTMLButtonElement>("#up-check")?.addEventListener("click", async () => {
      const msg = vSection.querySelector<HTMLElement>("#up-msg")!;
      msg.textContent = "Recherche…";
      const info = await api.CheckForUpdate();
      msg.textContent = info.available ? `Vinyl ${info.version} est disponible.` : "Vinyl est à jour.";
      announce(info);
    });
    vSection.querySelector<HTMLButtonElement>("#beta-on")?.addEventListener("click", async () => {
      const code = vSection.querySelector<HTMLInputElement>("#beta-code")!.value;
      try {
        await api.SetBetaCode(code);
        renderVersion();
        api.CheckForUpdate().then(announce).catch(() => {});
      } catch {
        vSection.querySelector<HTMLElement>("#beta-msg")!.textContent = "Code invalide";
      }
    });
    vSection.querySelector<HTMLButtonElement>("#beta-off")?.addEventListener("click", async () => {
      await api.SetBetaCode("");
      renderVersion();
      api.CheckForUpdate().then(announce).catch(() => {}); // une bêta déjà proposée ne l'est plus
    });
  };
  renderVersion();

  const renderSpotify = async () => {
    if (!cur) return;
    let connected = await api.SpotifyStatus();
    section.innerHTML = `
      <h2>Spotify</h2>
      <p class="text-[14px] leading-relaxed" style="color:rgba(42,29,27,.8)">La platine pilote Spotify sans connexion.
        Connecte ton compte pour retrouver ta bibliothèque (albums, playlists, titres likés) : Spotify te demandera d'autoriser Vinyl.</p>
      <div class="flex items-center gap-3">
        <button id="sp-btn" class="btn-dark cursor-pointer">${connected ? "Se déconnecter" : "Se connecter à Spotify"}</button>
        <p id="sp-msg" class="text-[14px]" style="color:rgba(42,29,27,.8)">${connected ? "Connecté." : ""}</p>
      </div>
      <details class="text-[14px]" style="color:rgba(42,29,27,.8)" ${cur.spotifyClientId ? "open" : ""}>
        <summary class="cursor-pointer">Utiliser ma propre app Spotify (si Spotify refuse l'accès à ta bibliothèque)</summary>
        <p class="mt-2 leading-relaxed">L'app Vinyl n'ouvre la bibliothèque qu'aux comptes inscrits par son auteur. Avec ta propre
          app Spotify (gratuite, 3 minutes), tu y as accès sans inscription.</p>
        <div id="sp-wizard"></div>
      </details>`;
    const msg = section.querySelector<HTMLElement>("#sp-msg")!;
    const btn = section.querySelector<HTMLButtonElement>("#sp-btn")!;
    mountSpotifyAppWizard(section.querySelector<HTMLElement>("#sp-wizard")!, {
      clientId: cur.spotifyClientId,
      openURL: (u) => void api.OpenURL(u),
      copy: (t) => Clipboard.SetText(t),
      save: async (id) => {
        if (id === cur?.spotifyClientId) return;
        if (connected) await api.SpotifyLogout(); // le jeton appartient à l'autre app
        connected = false;
        cur = await api.UpdateSettings({ spotifyClientId: id });
        btn.textContent = "Se connecter à Spotify";
        msg.textContent = "";
      },
      connect: async () => {
        if (connected) await api.SpotifyLogout();
        await api.SpotifyLogin();
        connected = true;
        btn.textContent = "Se déconnecter";
        msg.textContent = "Connecté.";
      },
      verify: () => api.Library("spotify", "playlists", 0),
    });
    btn.onclick = async () => {
      btn.disabled = true;
      try {
        if (connected) await api.SpotifyLogout();
        else {
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
