// Bouton « + » du tableau : ajoute le morceau en cours à une playlist de l'utilisateur. Un menu liste ses
// playlists modifiables (chargées à la première ouverture, gardées par source) ; un clic ajoute le
// morceau et un message le confirme (ou dit qu'il y était déjà).
import type { Item, NowPlaying } from "../api";
import { escapeHtml } from "../views/grid";

export interface AddToPlaylistDeps {
  playlists(source: string): Promise<Item[]>;
  /** false : le morceau y était déjà (rien d'ajouté). */
  add(np: NowPlaying, playlistId: string): Promise<boolean>;
}

const PLUS = '<svg viewBox="0 0 24 24" width="16" height="16" aria-hidden="true"><path d="M4 6h11M4 11h11M4 16h7M18 13v8M14 17h8"/></svg>';
const FILTER_FROM = 8; // champ de recherche au-delà de ce nombre de playlists

export interface AddToPlaylist { update(np: NowPlaying | null): void; close(): void; destroy(): void; }

export function mountAddToPlaylist(btn: HTMLButtonElement, menu: HTMLElement, deps: AddToPlaylistDeps): AddToPlaylist {
  btn.innerHTML = PLUS;
  btn.setAttribute("aria-label", "Ajouter à une playlist");
  btn.title = "Ajouter à une playlist";
  btn.setAttribute("aria-haspopup", "menu");
  btn.setAttribute("aria-expanded", "false");
  menu.hidden = true;
  menu.setAttribute("role", "menu");
  const cache = new Map<string, Promise<Item[]>>();
  let np: NowPlaying | null = null;
  let closeTimer = 0;
  let gen = 0;

  const status = (msg: string) => {
    const s = menu.querySelector<HTMLElement>(".vy-pl-status");
    if (s) s.textContent = msg;
  };
  const close = () => {
    clearTimeout(closeTimer);
    gen++;
    menu.hidden = true;
    btn.setAttribute("aria-expanded", "false");
  };
  const render = (items: Item[]) => {
    menu.innerHTML = `
      <p class="vy-pl-title">Ajouter à…</p>
      ${items.length > FILTER_FROM ? '<input class="vy-pl-filter" type="search" placeholder="Chercher" aria-label="Chercher une playlist">' : ""}
      <div class="vy-pl-list">${items.map((it) => `<button role="menuitem" data-id="${escapeHtml(it.id)}">${escapeHtml(it.title)}</button>`).join("")}</div>
      <p class="vy-pl-status" role="status">${items.length ? "" : "Aucune playlist modifiable"}</p>`;
    const filter = menu.querySelector<HTMLInputElement>(".vy-pl-filter");
    filter?.addEventListener("input", () => {
      const q = filter.value.trim().toLowerCase();
      menu.querySelectorAll<HTMLElement>(".vy-pl-list button").forEach((b) => (b.hidden = !!q && !b.textContent!.toLowerCase().includes(q)));
    });
    (filter ?? menu.querySelector<HTMLElement>(".vy-pl-list button"))?.focus();
  };
  const open = () => {
    if (!np) return;
    const cur = np, g = ++gen;
    menu.hidden = false;
    btn.setAttribute("aria-expanded", "true");
    menu.innerHTML = '<p class="vy-pl-status" role="status">…</p>';
    let list = cache.get(cur.source);
    if (!list) {
      list = deps.playlists(cur.source);
      cache.set(cur.source, list);
      list.catch(() => cache.delete(cur.source)); // nouvel essai à la prochaine ouverture
    }
    list.then((items) => { if (g === gen) render(items); })
      .catch((err) => { if (g === gen) status(String(err).replace(/^Error: /, "")); });
  };

  btn.onclick = (e) => {
    e.stopPropagation();
    if (menu.hidden) open(); else close();
  };
  menu.addEventListener("click", (e) => {
    e.stopPropagation();
    const b = (e.target as HTMLElement).closest<HTMLButtonElement>("button[data-id]");
    if (!b || !np) return;
    const g = gen, name = b.textContent ?? "";
    menu.querySelectorAll<HTMLButtonElement>(".vy-pl-list button").forEach((x) => (x.disabled = true));
    status("Ajout…");
    deps.add(np, b.dataset.id!)
      .then((added) => {
        if (g !== gen) return;
        status(added ? `Ajouté à « ${name} »` : `Déjà dans « ${name} »`);
        closeTimer = window.setTimeout(close, 1400);
      })
      .catch((err) => {
        if (g !== gen) return;
        status(String(err).replace(/^Error: /, ""));
        menu.querySelectorAll<HTMLButtonElement>(".vy-pl-list button").forEach((x) => (x.disabled = false));
      });
  });
  const onDoc = (e: Event) => { if (!menu.hidden && !menu.contains(e.target as Node) && e.target !== btn) close(); };
  const onKey = (e: KeyboardEvent) => { if (e.key === "Escape" && !menu.hidden) close(); };
  document.addEventListener("pointerdown", onDoc);
  document.addEventListener("keydown", onKey);

  return {
    update(next) {
      const changed = (next?.source ?? "") + (next?.trackId ?? "") !== (np?.source ?? "") + (np?.trackId ?? "");
      np = next;
      btn.hidden = !next;
      if (changed) close(); // le menu visait le morceau précédent
    },
    close,
    destroy() {
      close();
      document.removeEventListener("pointerdown", onDoc);
      document.removeEventListener("keydown", onKey);
    },
  };
}
