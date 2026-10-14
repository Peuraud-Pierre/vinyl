// Cycle du bouton « Répéter » : aucun → tout → un seul → aucun.
// Spotify ne connaît que oui/non : aucun ↔ tout.
export function nextRepeat(mode: string, source = "applemusic"): "off" | "all" | "one" {
  if (source === "spotify") return mode === "off" ? "all" : "off";
  return mode === "all" ? "one" : mode === "one" ? "off" : "all";
}
