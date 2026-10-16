// Grande table de la platine : plateau (où reposent platine et pile), ceinture avec un tiroir au milieu,
// puis l'ombre sous la table et deux pieds. Les tailles suivent l'échelle de la scène (--k).
export function mountRoomTable(el: HTMLElement): { setScale(k: number): void } {
  el.classList.add("vy-room-table");
  el.innerHTML = `
    <div class="vy-rt-top"></div>
    <div class="vy-rt-apron"><div class="vy-rt-drawer"><span class="vy-rt-knob"></span></div></div>
    <div class="vy-rt-leg vy-rt-leg-l"></div>
    <div class="vy-rt-leg vy-rt-leg-r"></div>`;
  return { setScale(k) { el.style.setProperty("--k", String(k)); } };
}
