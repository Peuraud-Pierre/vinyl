import { Events } from "@wailsio/runtime";
import { VinylService } from "../bindings/vinyl/internal/app";

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

export const api = VinylService;

export function onSnapshot(cb: (s: Snapshot) => void): () => void {
  return Events.On("nowplaying", (ev) => cb(ev.data as Snapshot));
}
export function onSettings(cb: (s: Settings) => void): () => void {
  return Events.On("settings", (ev) => cb(ev.data as Settings));
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
