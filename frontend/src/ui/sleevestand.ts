// Support de pochette « en écoute » (comme les présentoirs en chêne « Now Playing ») : la pochette du
// morceau en cours est calée dans sa rainure ; le rebord avant porte l'inscription gravée.
export function mountSleeveStand(el: HTMLElement): { setScale(k: number): void } {
  el.classList.add("vy-stand");
  el.setAttribute("aria-hidden", "true");
  const groove = Object.assign(document.createElement("div"), { className: "vy-stand-groove" });
  const front = Object.assign(document.createElement("div"), { className: "vy-stand-front", textContent: "EN ÉCOUTE" });
  el.replaceChildren(front, groove);
  return { setScale(k) { el.style.setProperty("--k", String(k)); } };
}
