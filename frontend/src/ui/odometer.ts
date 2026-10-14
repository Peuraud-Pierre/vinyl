// Compteur à rouleaux : chaque chiffre est une colonne 0–9 qui défile verticalement.
export function odometerDigits(n: number, width = 8): number[] {
  const max = 10 ** width - 1;
  const v = Number.isFinite(n) ? Math.min(max, Math.max(0, Math.floor(n))) : 0;
  return String(v).padStart(width, "0").split("").map(Number);
}

export function mountOdometer(el: HTMLElement, width = 8): (n: number) => void {
  const reel = Array.from({ length: 10 }, (_, d) => `<span>${d}</span>`).join("");
  el.className = "vy-odo";
  el.innerHTML = Array.from({ length: width }, () => `<div class="vy-odo-cell"><div class="vy-odo-reel">${reel}</div></div>`).join("");
  const reels = Array.from(el.querySelectorAll<HTMLElement>(".vy-odo-reel"));
  return (n) => odometerDigits(n, width).forEach((d, i) => { reels[i]!.style.transform = `translateY(${-d}em)`; });
}
