import { Events } from "@wailsio/runtime";
import { VinylService } from "../bindings/vinyl/internal/app";
import type { WidgetStatus } from "./widgetupdate";
import { createOptimistic } from "./optimistic";

// Types volontairement « larges » (string, string[]) pour rester compatibles avec les interfaces
// générées par `wails3 generate bindings` ; les valeurs possibles sont commentées.
export interface NowPlaying {
  source: string; trackId: string; title: string; artist: string; album: string;
  durationMs: number; positionMs: number; state: string /* playing | paused | stopped */; artworkUrl: string;
}
export interface Snapshot { nowPlaying: NowPlaying; problem: string /* "" | no-player | permission */; problemApp: string; }
export interface Settings {
  source: string;  // auto | applemusic | spotify
  texture: string; // black | picture | yolk | split | solid
  discColors: string[]; accent: string; crackle: boolean;
  widgetX: number; widgetY: number; widgetPlaced: boolean; spotifyClientId: string;
}

// Morceau posé sur la pile des derniers écoutés (à droite du disque).
export interface HistoryEntry { source: string; trackId: string; title: string; artist: string; album: string; artworkUrl: string; }

export const api = VinylService;

// États du lecteur : tous les abonnés passent par le filtre de lecture/pause instantanée (optimistic.ts).
const optimistic = createOptimistic();
const snapSubs = new Set<(s: Snapshot) => void>();
let lastSnap: Snapshot | undefined;
let listening = false;
const publish = (s: Snapshot) => { lastSnap = s; snapSubs.forEach((cb) => cb(s)); };

export function onSnapshot(cb: (s: Snapshot) => void): () => void {
  if (!listening) {
    listening = true;
    Events.On("nowplaying", (ev) => publish(optimistic.filter(ev.data as Snapshot)));
    api.GetSnapshot().then((s) => { lastSnap ??= s; }).catch(() => {}); // pour que le premier clic soit instantané
  }
  snapSubs.add(cb);
  return () => { snapSubs.delete(cb); };
}

// Lecture/pause : le disque, le titre et les boutons réagissent au clic, sans attendre le lecteur.
export function playPause(): Promise<void> {
  const o = optimistic.toggle(lastSnap);
  if (o) publish(o);
  return api.PlayPause().catch((e) => {
    optimistic.cancel();
    api.GetSnapshot().then(publish).catch(() => {});
    throw e;
  });
}
export function onSettings(cb: (s: Settings) => void): () => void {
  return Events.On("settings", (ev) => cb(ev.data as Settings));
}
export function onHistory(cb: (l: HistoryEntry[]) => void): () => void {
  return Events.On("history", (ev) => cb((ev.data as HistoryEntry[] | null) ?? []));
}
export function onWidgetUpdate(cb: (s: WidgetStatus) => void): () => void {
  return Events.On("widgetupdate", (ev) => cb(ev.data as WidgetStatus));
}
export function applyAccent(color: string): void {
  document.documentElement.style.setProperty("--accent", color);
}

export interface Item { id: string; kind: string; title: string; subtitle: string; artworkUrl: string; url: string; }
export interface Page { items: Item[] | null; next: number; }
export interface ArtistPage { source: string; name: string; imageUrl: string; url: string; topTracks: Item[] | null; albums: Item[] | null; }

export async function activeSource(): Promise<"applemusic" | "spotify"> {
  const snap = await api.GetSnapshot();
  if (snap.nowPlaying.source === "spotify" || snap.nowPlaying.source === "applemusic") return snap.nowPlaying.source;
  const s = await api.GetSettings();
  return s.source === "spotify" ? "spotify" : "applemusic";
}

export interface StatsSnapshot { count: number; since: string; }
export function onStats(cb: (s: StatsSnapshot) => void): () => void {
  return Events.On("stats", (ev) => cb(ev.data as StatsSnapshot));
}

export interface Track { id: string; number: number; title: string; artist: string; album: string; durationMs: number; loved: boolean; }
export interface Collection { id: string; kind: string; title: string; artist: string; genre: string; year: string; artworkUrl: string; source: string; tracks: Track[] | null; }

export interface QueueState { shuffle: boolean; repeat: string; upcoming: Track[] | null; note: string; }
