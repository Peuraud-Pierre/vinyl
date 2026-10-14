// Durée façon MD Vinyl : 3′37″, ou 1h02′05″ au-delà d'une heure.
export function formatDuration(ms: number): string {
  if (!Number.isFinite(ms) || ms < 0) return "";
  const total = Math.floor(ms / 1000);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = String(total % 60).padStart(2, "0");
  return h > 0 ? `${h}h${String(m).padStart(2, "0")}′${s}″` : `${m}′${s}″`;
}
