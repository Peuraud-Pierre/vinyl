// Étagère façon MD Vinyl (cf. screen/playlistvisual.jpg) : coverflow 3D avec tranches nommées,
// ou pile de pochettes. Un clic ouvre la page de l'album ou de la playlist.
import { activeSource, api, type Item } from "../api";
import { albumParam, go, TABS, type Tab } from "../router";
import { icons } from "../ui/icons";
import { imageTint } from "../ui/theme";
import { clampOffset, SHELF_ANGLE, SHELF_GAP, SHELF_PERSPECTIVE, shelfTransform, smoothStep, visibleIndices } from "./coverflow";
import { albumCoverRect, flightFrames } from "../ui/flight";
import { escapeHtml, initials } from "./grid";
import { filterItems, spineColors } from "./shelf";
import { recall, remember } from "./shelfcache";

const SEGMENTS: Tab[] = ["albums", "playlists"];
const TAB_LABELS: Record<Tab, string> = { albums: "Albums", playlists: "Playlists", artistes: "Artistes", likes: "Titres likés" };
const KIND: Record<Tab, string> = { albums: "albums", playlists: "playlists", artistes: "artists", likes: "likes" };
type Mode = "side" | "flat";

const readMode = (): Mode => {
  try { return localStorage.getItem("vinyl.shelf") === "flat" ? "flat" : "side"; } catch { return "side"; }
};
const saveMode = (m: Mode) => { try { localStorage.setItem("vinyl.shelf", m); } catch { /* stockage indisponible */ } };
const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;

export function mount(el: HTMLElement, tabParam: string): () => void {
  let tab = (TABS as readonly string[]).includes(tabParam) ? (tabParam as Tab) : "albums";
  let alive = true;
  let mode = readMode();
  let query = "";
  el.innerHTML = `
    <section class="relative h-screen">
      <div id="sh-area" class="absolute" style="left:0;right:0;top:0;bottom:0"></div>
      <p id="sh-status" class="absolute left-0 text-center text-[15px]" style="right:130px;bottom:18px;color:var(--ink-soft)"></p>
      <button id="sh-menu" class="glass round-btn vy-drop fixed z-30" style="left:92px;top:28px" aria-label="Menu de l'étagère" aria-expanded="false">${icons.shelf}</button>
      <div id="sh-menu-pop" class="glass fixed z-30 hidden w-64 rounded-[30px] p-2" style="left:92px;top:88px">
        <button data-nav="platine" class="menu-item">${icons.mini} Platine</button>
        <button data-nav="artistes" class="menu-item">${icons.person} Artistes</button>
        <button data-nav="likes" class="menu-item">${icons.play} Titres likés</button>
        <button id="sh-mode" class="menu-item"></button>
      </div>
      <div id="sh-tabs" class="glass vy-slide fixed z-30 flex gap-1 rounded-[30px] p-1" style="left:calc(50% - 65px);top:30px;translate:-50% 0" role="tablist" aria-label="Type de collection">
        ${SEGMENTS.map((t) => `<button role="tab" data-tab="${t}" aria-selected="${t === tab}" class="pill h-11 cursor-pointer px-6 text-[17px] font-medium ${t === tab ? "pill-on" : ""}">${TAB_LABELS[t]}</button>`).join("")}
      </div>
      <h2 id="sh-title" class="glass vy-slide fixed z-30 hidden rounded-[30px] px-6 py-2.5 text-[17px] font-semibold" style="left:calc(50% - 65px);top:30px;translate:-50% 0"></h2>
      <div class="vy-drop fixed z-30 flex items-center gap-3" style="right:51px;top:28px">
        <input id="sh-q" class="field glass hidden h-10 w-60" placeholder="Rechercher" aria-label="Rechercher dans la bibliothèque">
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
  let offset = 0; // défilement affiché de l'étagère (px, avant mise à l'échelle)
  let scrollGoal = 0; // défilement visé : la position affichée le rattrape image par image
  let raf = 0;
  let lastFrame = 0;
  let scale = 1;
  let entering = true; // première apparition : les boîtiers arrivent en cascade
  let scroller: HTMLElement | null = null;

  const paintHeader = () => {
    $("sh-mode").innerHTML = mode === "side" ? `${icons.stack} Vue pile` : `${icons.shelf} Vue étagère`;
    const segmented = SEGMENTS.includes(tab);
    $("sh-tabs").classList.toggle("hidden", !segmented);
    $("sh-title").classList.toggle("hidden", segmented);
    $("sh-title").textContent = TAB_LABELS[tab];
    el.querySelectorAll<HTMLElement>("[data-tab]").forEach((b) => {
      const on = b.dataset.tab === tab;
      b.classList.toggle("pill-on", on);
      b.setAttribute("aria-selected", String(on));
    });
  };

  const spine = (it: Item) => spineColors(it.title, tints.get(it.id));
  const shown = () => filterItems(items, query);

  // Pochettes décodées en arrière-plan (img.decode) avant d'être posées : aucun décodage pendant
  // le défilement. Une pochette déjà décodée s'affiche tout de suite.
  const decoded = new Set<string>();
  const applyCover = (face: HTMLElement, url: string) => {
    if (!url) return;
    const show = () => {
      face.style.backgroundImage = cssUrl(url);
      face.querySelector("span")?.remove();
    };
    if (decoded.has(url)) return show();
    face.style.opacity = "0";
    const img = new Image();
    img.src = url;
    img.decode().catch(() => {}).then(() => {
      decoded.add(url);
      if (!alive) return;
      show();
      face.animate([{ opacity: 0 }, { opacity: 1 }], { duration: 220, easing: "ease-out" });
      face.style.opacity = "";
    });
  };

  const artStyle = (it: Item) => {
    const c = spine(it);
    return `background-color:${c.bg};` + (it.artworkUrl ? `background-image:${escapeHtml(cssUrl(it.artworkUrl))}` : "");
  };
  const word = (it: Item) =>
    it.artworkUrl ? "" : `<span class="absolute left-6 top-6 text-[44px] font-black" style="color:${spine(it).ink}">${escapeHtml(initials(it.title))}</span>`;
  const label = (it: Item) => `${escapeHtml(it.title)}${it.subtitle ? ` — ${escapeHtml(it.subtitle)}` : ""}`;

  // Boîtier du coverflow : face (pochette) + tranche nommée, reflet sur la face.
  const boxHtml = (it: Item) => {
    const c = spine(it);
    return `<button class="vy-cf" data-id="${escapeHtml(it.id)}" aria-label="${label(it)}">
      <div class="vy-face" style="background-color:${spine(it).bg}">${word(it)}</div>
      <div class="vy-spine" style="background:${c.bg};color:${c.ink}"><span><b>${escapeHtml(it.title)}</b>  ${escapeHtml(it.subtitle)}</span></div></button>`;
  };
  const MAX_STAGGER = 18;
  const pileHtml = (it: Item, i: number, animate: boolean) => {
    const anim = animate ? `vy-rise" style="--i:${Math.min(i, MAX_STAGGER)}` : "";
    return `<button class="vy-pile ${anim}" data-id="${escapeHtml(it.id)}" aria-label="${label(it)}"><div class="vy-cover" style="${artStyle(it)}">${word(it)}</div>
      <div class="vy-edge"><b>${escapeHtml(it.title)}</b><span class="opacity-80">${escapeHtml(it.subtitle)}</span></div></button>`;
  };

  // Étagère : tous les boîtiers vus par la tranche ; seuls ceux à l'écran sont dans le DOM.
  const halfWidth = () => area.clientWidth / 2 / scale;
  // Zone de droite à laisser libre : barre de commandes (≈ 130 px) et fondu du bord.
  const rightReserve = () => 170 / scale;
  const layout = () => {
    const row = area.querySelector<HTMLElement>("#cf-row");
    if (!row) return;
    const list = shown();
    const half = halfWidth();
    offset = clampOffset(offset, list.length, half, rightReserve());
    const [a, b] = visibleIndices(offset, list.length, half);
    const keep = new Set<string>();
    for (let i = a; i <= b; i++) {
      const it = list[i]!;
      keep.add(it.id);
      let slot = row.querySelector<HTMLElement>(`[data-id="${CSS.escape(it.id)}"]`);
      if (!slot) {
        row.insertAdjacentHTML("beforeend", boxHtml(it));
        slot = row.lastElementChild as HTMLElement;
        applyCover(slot.querySelector<HTMLElement>(".vy-face")!, it.artworkUrl);
        if (entering) {
          slot.classList.add("vy-shelf-in");
          slot.style.setProperty("--i", String(Math.max(0, i - a)));
        }
      }
      const t = shelfTransform(i, offset, half);
      slot.style.transform = `translate(-50%,-50%) translateX(${t.x}px) rotateY(${t.rotateY}deg)`;
      slot.style.zIndex = String(t.zIndex);
      slot.classList.toggle("cf-right", t.spineRight);
    }
    row.querySelectorAll<HTMLElement>("[data-id]").forEach((n) => { if (!keep.has(n.dataset.id!)) n.remove(); });
    if (list.length > 0) entering = false;
    if (!query && b >= list.length - 6) load();
  };

  const memo = () => remember(`${source}:${tab}`, { items, next, position: offset });

  // Défilement fluide : une seule mise en page par image (requestAnimationFrame), aucune transition CSS.
  const frame = (now: number) => {
    const dt = lastFrame ? now - lastFrame : 16;
    lastFrame = now;
    offset = smoothStep(offset, scrollGoal, dt);
    layout();
    if (offset !== scrollGoal) raf = requestAnimationFrame(frame);
    else { raf = 0; lastFrame = 0; }
  };
  const scrollBy = (dx: number) => {
    scrollGoal = clampOffset(scrollGoal + dx, shown().length, halfWidth(), rightReserve());
    if (!raf && scrollGoal !== offset) raf = requestAnimationFrame(frame);
  };

  // Mise à l'échelle : boîtiers d'environ 80 % de la hauteur, comme dans la vidéo.
  const fit = () => {
    const row = area.querySelector<HTMLElement>("#cf-row");
    if (!row) return;
    scale = Math.max(0.8, (area.clientHeight * 0.8) / 300); // vidéo : boîtiers sur ~80 % de la hauteur
    row.style.scale = String(scale);
    // La profondeur grandit avec l'échelle : sans ça, la perspective serait bien plus forte que dans la vidéo.
    (row.parentElement as HTMLElement).style.perspective = `${SHELF_PERSPECTIVE * scale}px`;
    layout();
  };
  const ro = new ResizeObserver(fit);
  ro.observe(area);

  const render = (animate = false) => {
    if (mode === "side") {
      area.innerHTML = `<div id="cf" class="relative h-full w-full overflow-hidden" style="perspective-origin:50% 50%;contain:layout paint">
        <div id="cf-row" class="absolute left-1/2 top-1/2" style="transform-style:preserve-3d"></div>
        <div class="pointer-events-none absolute inset-y-0 left-0 w-[7%]" style="background:linear-gradient(90deg,color-mix(in srgb, var(--tint) 35%, #0a0605),transparent)"></div>
        <div class="pointer-events-none absolute inset-y-0 right-0 w-[7%]" style="background:linear-gradient(270deg,color-mix(in srgb, var(--tint) 35%, #0a0605),transparent)"></div></div>`;
      scroller = null;
      // Défilement continu à la molette ou au trackpad (horizontal ou vertical).
      area.querySelector("#cf")!.addEventListener("wheel", (e) => {
        const w = e as WheelEvent;
        scrollBy((Math.abs(w.deltaX) > Math.abs(w.deltaY) ? w.deltaX : w.deltaY) / scale);
        w.preventDefault();
      }, { passive: false });
      fit();
      return;
    }
    const html = shown().map((it, i) => pileHtml(it, i, animate)).join("");
    area.innerHTML = `<div id="sh-scroll" class="shelf-scroll h-full overflow-y-auto"><div class="flex flex-col py-[80px]" style="padding-left:160px;padding-bottom:300px">${html}</div></div>`;
    scroller = $("sh-scroll");
    scroller.addEventListener("scroll", () => {
      if (!scroller || query) return;
      if (scroller.scrollHeight - scroller.scrollTop - scroller.clientHeight < 600) load();
    }, { passive: true });
  };

  const recolor = (it: Item) => {
    const c = spine(it);
    area.querySelectorAll<HTMLElement>(`[data-id="${CSS.escape(it.id)}"] .vy-cover, [data-id="${CSS.escape(it.id)}"] .vy-face`).forEach((n) => {
      n.style.backgroundColor = c.bg;
    });
    area.querySelectorAll<HTMLElement>(`[data-id="${CSS.escape(it.id)}"] .vy-spine`).forEach((n) => {
      n.style.background = c.bg;
      n.style.color = c.ink;
    });
  };

  const decorate = async (it: Item) => {
    if (!it.artworkUrl && source === "applemusic" && it.kind !== "artist") {
      it.artworkUrl = await api.ItemArtwork(source, it.id).catch(() => "");
      if (!alive || !it.artworkUrl) return;
      area.querySelectorAll<HTMLElement>(`[data-id="${CSS.escape(it.id)}"] .vy-cover, [data-id="${CSS.escape(it.id)}"] .vy-face`).forEach((n) => applyCover(n, it.artworkUrl));
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
    if (items.length === 0) status.innerHTML = `<span class="vy-spin" role="status" aria-label="Chargement"></span>`;
    try {
      const page = await api.Library(source, KIND[tab], next);
      if (!alive) return;
      const fresh = page.items ?? [];
      const first = items.length === 0;
      items.push(...fresh);
      next = page.next;
      if (mode === "side") {
        if (!area.querySelector("#cf-row")) render();
        else layout();
      } else {
        const keep = scroller?.scrollTop ?? 0;
        render(first);
        scroller!.scrollTop = keep;
      }
      fresh.forEach(decorate);
      memo();
      status.textContent = items.length === 0 ? "Rien ici pour l'instant." : "";
    } catch (e) {
      showError(e);
    } finally {
      loading = false;
    }
  }

  const open = async (it: Item) => {
    if (it.kind === "artist") return go("artiste", it.title);
    if (it.kind === "track") {
      try {
        await api.PlayItem(source, it.id);
        go("platine");
      } catch (err) {
        showError(err);
      }
      return;
    }
    flyToAlbum(it);
    go("album", albumParam(source, it.id));
  };

  // La pochette cliquée s'envole jusqu'à sa place sur la page album (vidéo MD Vinyl, 20–21 s).
  const flyToAlbum = (it: Item) => {
    const face = area.querySelector<HTMLElement>(`[data-id="${CSS.escape(it.id)}"] .vy-face`);
    if (!face || !it.artworkUrl || mode !== "side") return;
    const r = face.getBoundingClientRect();
    const slot = face.closest<HTMLElement>("[data-id]")!;
    const angle = slot.classList.contains("cf-right") ? -SHELF_ANGLE : SHELF_ANGLE; // même angle que le boîtier
    const ghost = document.createElement("div");
    ghost.className = "pointer-events-none fixed z-[60] bg-cover bg-center";
    ghost.style.backgroundImage = cssUrl(it.artworkUrl);
    ghost.style.boxShadow = "0 24px 50px rgba(0,0,0,.45)";
    document.body.appendChild(ghost);
    const to = albumCoverRect(window.innerWidth);
    const flight = ghost.animate(flightFrames({ left: r.left, top: r.top, width: r.width, height: r.height }, to, angle), {
      duration: 520, easing: "cubic-bezier(.2,.75,.2,1)", fill: "forwards",
    });
    flight.finished.then(() => ghost.animate([{ opacity: 1 }, { opacity: 0 }], { duration: 260, delay: 220, fill: "forwards" }).finished)
      .finally(() => ghost.remove());
  };

  area.addEventListener("animationend", (e) => (e.target as HTMLElement).classList.remove("vy-fly", "vy-rise", "vy-shelf-in"));
  area.addEventListener("click", (e) => {
    const btn = (e.target as HTMLElement).closest<HTMLElement>("[data-id]");
    const it = btn && items.find((i) => i.id === btn.dataset.id);
    if (!it || !btn) return;
    memo();
    open(it);
  });

  const onKey = (e: KeyboardEvent) => {
    const t = e.target;
    if ((t instanceof Element && t.closest("input, select, textarea")) || mode !== "side") return;
    if (e.key === "ArrowLeft") { scrollBy(-SHELF_GAP); e.stopPropagation(); e.preventDefault(); }
    if (e.key === "ArrowRight") { scrollBy(SHELF_GAP); e.stopPropagation(); e.preventDefault(); }
  };
  window.addEventListener("keydown", onKey, true); // avant les raccourcis précédent/suivant de main.ts

  // Changement d'onglet sur place : les pochettes s'effacent en flou, les nouvelles arrivent.
  const switchTab = async (target: Tab) => {
    if (target === tab || loading) return;
    tab = target;
    history.replaceState(null, "", `#/bibliotheque/${tab}`);
    paintHeader();
    area.firstElementChild?.classList.add("vy-leave");
    await new Promise((r) => setTimeout(r, 220));
    if (!alive || tab !== target) return;
    items.length = 0;
    tints.clear();
    next = 0;
    offset = scrollGoal = 0;
    entering = true;
    area.innerHTML = "";
    scroller = null;
    status.textContent = "";
    if (tab === "likes" && source !== "spotify") {
      status.textContent = "Les titres likés sont disponibles avec Spotify.";
      return;
    }
    load();
  };

  const toggleMenu = (force?: boolean) => {
    const open = force ?? $("sh-menu-pop").classList.contains("hidden");
    $("sh-menu-pop").classList.toggle("hidden", !open);
    $("sh-menu").setAttribute("aria-expanded", String(open));
  };
  $("sh-menu").onclick = (e) => { e.stopPropagation(); toggleMenu(); };
  const closeMenu = (e: MouseEvent) => { if (!(e.target as HTMLElement).closest("#sh-menu-pop, #sh-menu")) toggleMenu(false); };
  document.addEventListener("click", closeMenu);
  el.querySelectorAll<HTMLElement>("[data-nav]").forEach((b) => (b.onclick = () => {
    toggleMenu(false);
    const nav = b.dataset.nav!;
    if (nav === "platine") go("platine");
    else switchTab(nav as Tab);
  }));
  el.querySelectorAll<HTMLElement>("[data-tab]").forEach((b) => (b.onclick = () => switchTab(b.dataset.tab as Tab)));
  $("sh-mode").onclick = () => {
    toggleMenu(false);
    mode = mode === "side" ? "flat" : "side";
    saveMode(mode);
    paintHeader();
    render(true);
  };
  $("sh-search").onclick = () => {
    const q = $<HTMLInputElement>("sh-q");
    q.classList.toggle("hidden");
    if (!q.classList.contains("hidden")) q.focus();
    else if (query) { query = ""; q.value = ""; offset = scrollGoal = 0; render(); }
  };
  $<HTMLInputElement>("sh-q").addEventListener("input", (e) => {
    query = (e.target as HTMLInputElement).value;
    offset = scrollGoal = 0;
    render();
    status.textContent = query && shown().length === 0 ? "Aucun résultat dans ce qui est chargé." : "";
  });

  paintHeader();
  activeSource().then((s) => {
    source = s;
    if (tab === "likes" && s !== "spotify") {
      status.textContent = "Les titres likés sont disponibles avec Spotify.";
      return;
    }
    const saved = recall(`${source}:${tab}`);
    if (saved && saved.items.length > 0) {
      items.push(...saved.items);
      next = saved.next;
      offset = scrollGoal = saved.position;
      entering = false;
      render();
      items.forEach(decorate);
      return;
    }
    load();
  });
  return () => {
    cancelAnimationFrame(raf);
    ro.disconnect();
    if (items.length > 0) memo();
    alive = false;
    window.removeEventListener("keydown", onKey, true);
    document.removeEventListener("click", closeMenu);
  };
}
