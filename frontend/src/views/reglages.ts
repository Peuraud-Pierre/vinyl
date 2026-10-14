import { api, type Settings } from "../api";

const escapeAttr = (s: string) => s.replace(/[&"<>]/g, (c) => ({ "&": "&amp;", '"': "&quot;", "<": "&lt;", ">": "&gt;" })[c]!);

export function mount(el: HTMLElement): () => void {
  el.innerHTML = `
    <div class="flex max-w-xl flex-col gap-6 p-10 pt-14 text-sm">
      <h1 class="text-xl font-semibold">Réglages</h1>
      <label class="flex items-center justify-between">Craquement quand l'aiguille se pose
        <input type="checkbox" id="crackle" class="h-4 w-4"></label>
      <section id="spotify-section"></section>
    </div>`;
  const crackle = el.querySelector<HTMLInputElement>("#crackle")!;
  const section = el.querySelector<HTMLElement>("#spotify-section")!;
  let cur: Settings | null = null;

  const renderSpotify = async () => {
    if (!cur) return;
    const connected = await api.SpotifyStatus();
    section.innerHTML = `
      <h2 class="mb-2 font-semibold">Spotify</h2>
      <p class="mb-3 text-neutral-400">Crée une app gratuite sur developer.spotify.com/dashboard, ajoute l'URI de redirection
        <code class="rounded bg-neutral-800 px-1">http://127.0.0.1:43821/callback</code>, puis colle son Client ID ici.</p>
      <div class="flex gap-2">
        <input id="cid" class="flex-1 rounded bg-neutral-800 px-2 py-1" placeholder="Client ID" value="${escapeAttr(cur.spotifyClientId)}">
        <button id="sp-btn" class="rounded px-3 py-1 text-neutral-900" style="background:var(--accent)">${connected ? "Se déconnecter" : "Se connecter"}</button>
      </div>
      <p id="sp-msg" class="mt-2 text-neutral-400">${connected ? "Connecté." : ""}</p>`;
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
    renderSpotify();
  });
  crackle.onchange = async () => {
    cur = await api.UpdateSettings({ crackle: crackle.checked });
  };
  return () => {};
}
