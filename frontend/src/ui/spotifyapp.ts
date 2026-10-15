// Assistant « ma propre app Spotify » : chacun crée son app sur le site Spotify (il y est autorisé
// d'office), colle son Client ID, puis Vinyl se connecte et vérifie l'accès à la bibliothèque. Évite
// d'inscrire chaque ami dans l'app Vinyl (mode développement de Spotify, 25 comptes au plus).
export const REDIRECT_URI = "http://127.0.0.1:43821/callback";
const CREATE_URL = "https://developer.spotify.com/dashboard/create";

export const isClientID = (s: string) => /^[0-9a-f]{32}$/i.test(s.trim());

export interface WizardDeps {
  clientId: string; // Client ID perso enregistré ("" : app Vinyl)
  openURL(u: string): void;
  copy(text: string): Promise<unknown>;
  save(clientId: string): Promise<unknown>;
  connect(): Promise<unknown>; // déconnexion éventuelle puis connexion avec le nouveau Client ID
  verify(): Promise<unknown>; // lecture de la bibliothèque : rejette avec la raison donnée par Spotify
}

const code = (t: string) => `<code class="rounded-md px-1.5" style="background:rgba(255,255,255,.55)">${t}</code>`;

export function mountSpotifyAppWizard(el: HTMLElement, deps: WizardDeps): void {
  el.innerHTML = `
    <ol class="mt-2 flex list-decimal flex-col gap-3 pl-5 leading-relaxed">
      <li>Crée ton app Spotify (connecte-toi avec ton compte Spotify) :
        <button data-open class="ml-1 cursor-pointer font-semibold underline">ouvrir la page de création</button>
        <ul class="mt-1 list-disc pl-5">
          <li>App name : ${code("Vinyl")} ; App description : ${code("Platine")}</li>
          <li>Redirect URIs : ${code(REDIRECT_URI)}
            <button data-copy class="ml-1 cursor-pointer underline">copier</button>, puis « Add »</li>
          <li>Coche « Web API », accepte les conditions, puis « Save »</li>
        </ul>
      </li>
      <li>Sur la page de ton app, copie le « Client ID » et colle-le ici :
        <div class="mt-2 flex gap-2">
          <label class="sr-only" for="cid">Client ID Spotify personnel</label>
          <input id="cid" data-cid class="field flex-1" placeholder="Client ID (32 caractères)" autocomplete="off" spellcheck="false" value="${deps.clientId.replace(/[^0-9a-fA-F]/g, "")}">
          <button data-go class="btn-dark cursor-pointer">Se connecter</button>
        </div>
      </li>
    </ol>
    <p data-msg class="mt-2" role="status"></p>
    ${deps.clientId ? `<button data-reset class="mt-1 cursor-pointer underline">Revenir à l'app Vinyl</button>` : ""}`;
  const msg = el.querySelector<HTMLElement>("[data-msg]")!;
  const input = el.querySelector<HTMLInputElement>("[data-cid]")!;
  const go = el.querySelector<HTMLButtonElement>("[data-go]")!;
  el.querySelector<HTMLButtonElement>("[data-open]")!.onclick = () => deps.openURL(CREATE_URL);
  el.querySelector<HTMLButtonElement>("[data-copy]")!.onclick = () =>
    void deps.copy(REDIRECT_URI).then(() => (msg.textContent = "Adresse copiée."), () => {});
  go.onclick = async () => {
    const id = input.value.trim();
    if (!isClientID(id)) {
      msg.textContent = "Le Client ID fait 32 caractères (chiffres et lettres de a à f).";
      return;
    }
    go.disabled = true;
    try {
      await deps.save(id);
      msg.textContent = "Termine la connexion dans ton navigateur…";
      await deps.connect();
      msg.textContent = "Vérification de l'accès à ta bibliothèque…";
      await deps.verify();
      msg.textContent = "C'est bon : ta bibliothèque Spotify est accessible.";
    } catch (e) {
      msg.textContent = `Ça n'a pas marché : ${e instanceof Error ? e.message : String(e)}. Vérifie l'adresse de retour et « Web API » dans ton app Spotify.`;
    } finally {
      go.disabled = false;
    }
  };
  el.querySelector<HTMLButtonElement>("[data-reset]")?.addEventListener("click", async () => {
    await deps.save("");
    msg.textContent = "L'app Vinyl sera utilisée à la prochaine connexion.";
  });
}
