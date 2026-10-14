// Cycle du bouton « Répéter » : aucun → tout → un seul → aucun.
export function nextRepeat(mode: string): "off" | "all" | "one" {
  return mode === "all" ? "one" : mode === "one" ? "off" : "all";
}
