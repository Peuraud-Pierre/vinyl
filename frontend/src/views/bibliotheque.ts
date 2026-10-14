// Étagère : pochettes debout vues par la tranche, ou empilées à plat.
import { activeSource, api, type Item } from "../api";
import { go, TABS, type Tab } from "../router";
import { icons } from "../ui/icons";
import { imageTint } from "../ui/theme";
import { escapeHtml, initials } from "./grid";
import { filterItems, spineColors } from "./shelf";

const TAB_LABELS: Record<Tab, string> = { albums: "Albums", playlists: "Playlists", artistes: "Artistes", likes: "Likés" };
const KIND: Record<Tab, string> = { albums: "albums", playlists: "playlists", artistes: "artists", likes: "likes" };
type Mode = "side" | "flat";

const readMode = (): Mode => {
  try { return localStorage.getItem("vinyl.shelf") === "flat" ? "flat" : "side"; } catch { return "side"; }
};
const saveMode = (m: Mode) => { try { localStorage.setItem("vinyl.shelf", m); } catch { /* stockage indisponible */ } };
const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;

export function mount(el: HTMLElement, tabParam: string): () => void {
  const tab = (TABS as readonly string[]).includes(tabParam) ? (tabParam as Tab) : "albums";
  let alive = true;
  let mode = readMode();
  let query = "";
  el.innerHTML = `
    <section class="relative h-screen">
      <div id="sh-area" class="absolute" style="left:0;right:150px;top:110px;bottom:56px"></div>
      <p id="sh-status" class="absolute left-0 text-center text-[15px]" style="right:150px;bottom:20px;color:var(--ink-soft)"></p>
      <button id="sh-back" class="glass round-btn fixed z-30" style="left:96px;top:30px" aria-label="Revenir à la platine">${icons.shelf}</button>
      <button id="sh-mode" class="glass round-btn fixed z-30" style="left:176px;top:30px"></button>
      <div class="glass fixed z-30 flex gap-1 rounded-[40px] p-[5px]" style="left:50%;top:32px;transform:translateX(-50%)" role="tablist" aria-label="Type de collection">
        ${TABS.map((t) => `<button role="tab" data-tab="${t}" aria-selected="${t === tab}" class="pill h-[50px] cursor-pointer text-[19px] font-medium ${t === tab ? "pill-on" : ""}">${TAB_LABELS[t]}</button>`).join("")}
      </div>
      <div class="fixed z-30 flex items-center gap-3" style="right:60px;top:30px">
        <input id="sh-q" class="field glass hidden w-64" placeholder="Rechercher" aria-label="Rechercher dans la bibliothèque">
        <button id="sh-search" class="glass round-btn" aria-label="Rechercher">${icons.search}</button>
      </div>
    </section>`;
  const $ = <T extends HTMLElement = HTMLElement>(id: string) => el.querySelector<T>("#" + id)!;
  const area = $("sh-area");
  const status = $("sh-status");
  const items: Item[] = [];
  const tints = new Map<string, string>();
  let source: "applemusic" | "spotify" = "applemusic";
  let next = 0;
  let loading = false;
  let scroller: HTMLElement | null = null;

  const paintModeButton = () => {
    $("sh-mode").innerHTML = mode === "side" ? icons.stack : icons.shelf;
    const label = mode === "side" ? "Passer en pile verticale" : "Passer en étagère";
    $("sh-mode").setAttribute("aria-label", label);
    $("sh-mode").title = label;
  };

  const spine = (it: Item) => spineColors(it.title, tints.get(it.id));

  const itemHtml = (it: Item) => {
    const c = spine(it);
    const art = it.artworkUrl ? `background-image:${escapeHtml(cssUrl(it.artworkUrl))}` : "";
    const word = it.artworkUrl ? "" : `<span class="absolute left-6 top-6 text-[44px] font-black" style="color:${c.ink}">${escapeHtml(initials(it.title))}</span>`;
    const label = `${escapeHtml(it.title)}${it.subtitle ? ` — ${escapeHtml(it.subtitle)}` : ""}`;
    if (mode === "side") {
      return `<button class="vy-slot" data-id="${escapeHtml(it.id)}" aria-label="${label}">
        <div class="vy-box"><div class="vy-face" style="${art}">${word}</div>
        <div class="vy-spine" style="background:${c.bg};color:${c.ink}"><span><b>${escapeHtml(it.title)}</b>  ${escapeHtml(it.subtitle)}</span></div></div></button>`;
    }
    return `<button class="vy-layer" data-id="${escapeHtml(it.id)}" aria-label="${label}">
      <div class="vy-flat" style="background:${c.bg};color:${c.ink}"><b class="text-[16px]">${escapeHtml(it.title)}</b><span class="text-[15px] opacity-85">${escapeHtml(it.subtitle)}</span></div>
      <div class="vy-sleeve" style="${art}">${word}</div></button>`;
  };

  const render = () => {
    const shown = filterItems(items, query);
    area.innerHTML = mode === "side"
      ? `<div id="sh-scroll" class="shelf-scroll flex h-full items-center overflow-x-auto overflow-y-hidden px-[220px]" style="perspective:1100px;perspective-origin:50% 40%">${shown.map(itemHtml).join("")}</div>`
      : `<div id="sh-scroll" class="shelf-scroll h-full overflow-y-auto"><div class="flex flex-col gap-1.5 py-[160px]" style="padding-left:120px">${shown.map(itemHtml).join("")}</div></div>`;
    scroller = $("sh-scroll");
    scroller.addEventListener("scroll", maybeLoad, { passive: true });
    if (mode === "side") {
      // La molette verticale fait défiler l'étagère de côté.
      scroller.addEventListener("wheel", (e) => {
        if (Math.abs(e.deltaY) > Math.abs(e.deltaX)) { scroller!.scrollLeft += e.deltaY; e.preventDefault(); }
      }, { passive: false });
    }
  };

  const recolor = (it: Item) => {
    const c = spine(it);
    area.querySelectorAll<HTMLElement>(`[data-id="${CSS.escape(it.id)}"] .vy-spine, [data-id="${CSS.escape(it.id)}"] .vy-flat`).forEach((n) => {
      n.style.background = c.bg;
      n.style.color = c.ink;
    });
  };

  const decorate = async (it: Item) => {
    if (!it.artworkUrl && source === "applemusic" && it.kind !== "artist") {
      it.artworkUrl = await api.ItemArtwork(source, it.id).catch(() => "");
      if (!alive || !it.artworkUrl) return;
      area.querySelectorAll<HTMLElement>(`[data-id="${CSS.escape(it.id)}"] .vy-face, [data-id="${CSS.escape(it.id)}"] .vy-sleeve`).forEach((n) => {
        n.style.backgroundImage = cssUrl(it.artworkUrl);
        n.innerHTML = "";
      });
    }
    const tint = await imageTint(it.artworkUrl);
    if (!alive || !tint) return;
    tints.set(it.id, tint);
    recolor(it);
  };

  const showError = (e: unknown) => {
    const msg = String(e);
    if (msg.includes("non connecté") || msg.includes("reconnecte")) {
      status.innerHTML = `Connecte ton compte Spotify pour voir ta bibliothèque. <button id="go-settings" class="underline">Ouvrir les Réglages</button>`;
      status.querySelector("#go-settings")!.addEventListener("click", () => go("reglages"));
    } else if (msg.includes("non disponible")) {
      status.textContent = "Non disponible pour Apple Music.";
    } else if (msg.includes("non lancé")) {
      status.textContent = "Ouvre l'app Musique pour voir ta bibliothèque.";
    } else if (msg.includes("non autorisée")) {
      status.innerHTML = `Vinyl n'a pas l'autorisation de piloter Musique. <button id="open-auto" class="underline">Ouvrir les réglages</button>`;
      status.querySelector("#open-auto")!.addEventListener("click", () => api.OpenAutomationSettings());
    } else {
      status.innerHTML = `${escapeHtml(msg)} <button id="retry" class="underline">Réessayer</button>`;
      status.querySelector("#retry")!.addEventListener("click", load);
    }
  };

  async function load() {
    if (loading || next < 0 || !alive) return;
    loading = true;
    status.textContent = "Chargement…";
    try {
      const page = await api.Library(source, KIND[tab], next);
      if (!alive) return;
      const fresh = page.items ?? [];
      items.push(...fresh);
      next = page.next;
      const keep = scroller ? [scroller.scrollLeft, scroller.scrollTop] : [0, 0];
      render();
      scroller!.scrollLeft = keep[0]!;
      scroller!.scrollTop = keep[1]!;
      fresh.forEach(decorate);
      status.textContent = items.length === 0 ? "Rien ici pour l'instant." : "";
    } catch (e) {
      showError(e);
    } finally {
      loading = false;
    }
  }

  function maybeLoad() {
    if (!scroller || query) return;
    const left = mode === "side"
      ? scroller.scrollWidth - scroller.scrollLeft - scroller.clientWidth
      : scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight;
    if (left < 600) load();
  }

  area.addEventListener("click", async (e) => {
    const btn = (e.target as HTMLElement).closest<HTMLElement>("[data-id]");
    const it = btn && items.find((i) => i.id === btn.dataset.id);
    if (!it) return;
    if (it.kind === "artist") return go("artiste", it.title);
    try {
      await api.PlayItem(source, it.id);
      go("platine");
    } catch (err) {
      showError(err);
    }
  });

  el.querySelectorAll<HTMLElement>("[data-tab]").forEach((b) => (b.onclick = () => go("bibliotheque", b.dataset.tab!)));
  $("sh-back").onclick = () => go("platine");
  $("sh-mode").onclick = () => {
    mode = mode === "side" ? "flat" : "side";
    saveMode(mode);
    paintModeButton();
    render();
  };
  $("sh-search").onclick = () => {
    const q = $<HTMLInputElement>("sh-q");
    q.classList.toggle("hidden");
    if (!q.classList.contains("hidden")) q.focus();
    else if (query) { query = ""; q.value = ""; render(); }
  };
  $<HTMLInputElement>("sh-q").addEventListener("input", (e) => {
    query = (e.target as HTMLInputElement).value;
    render();
    status.textContent = query && filterItems(items, query).length === 0 ? "Aucun résultat dans ce qui est chargé." : "";
  });

  paintModeButton();
  activeSource().then((s) => {
    source = s;
    if (tab === "likes" && s !== "spotify") {
      status.textContent = "Les titres likés sont disponibles avec Spotify.";
      return;
    }
    load();
  });
  return () => { alive = false; };
}
