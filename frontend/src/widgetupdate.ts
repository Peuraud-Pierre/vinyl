// Icône ☁️↓ de mise à jour du widget de bureau (rendu pur, testable sans Wails).
export interface WidgetStatus {
  state: string; // none | available | running | error
  reason: string;
  message: string;
}

const CLOUD = `<svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="1.8"
  stroke-linecap="round" stroke-linejoin="round"><path d="M7 18a4.5 4.5 0 0 1-.6-8.96A6 6 0 0 1 18 9.5a4 4 0 0 1-.5 8.5"/>
  <path d="M12 11v8m-3-3 3 3 3-3"/></svg>`;

export function render(el: HTMLElement, s: WidgetStatus, actions: { update: () => void }): void {
  el.replaceChildren();
  el.hidden = s.state === "none";
  if (el.hidden) return;

  const btn = document.createElement("button");
  btn.className = "flex h-8 w-8 items-center justify-center rounded-full bg-neutral-800/80 text-neutral-200 hover:bg-neutral-700";
  btn.dataset.state = s.state;
  if (s.state === "available") {
    btn.innerHTML = CLOUD;
    btn.title = s.message;
    btn.style.color = "var(--accent)";
    btn.onclick = actions.update;
  } else if (s.state === "running") {
    btn.innerHTML = CLOUD;
    btn.title = s.message || "Mise à jour du widget…";
    btn.classList.add("animate-spin");
    btn.disabled = true;
  } else {
    btn.textContent = "⚠️";
    btn.title = "La mise à jour du widget a échoué";
    btn.onclick = () => togglePopover(el, s.message, actions.update);
  }
  el.append(btn);
}

function togglePopover(el: HTMLElement, message: string, update: () => void): void {
  const open = el.querySelector("[data-popover]");
  if (open) {
    open.remove();
    return;
  }
  const pop = document.createElement("div");
  pop.dataset.popover = "";
  pop.className = "absolute right-0 top-10 w-80 rounded-lg bg-neutral-900 p-3 text-xs text-neutral-200 shadow-xl ring-1 ring-white/10";
  const msg = document.createElement("p");
  msg.className = "mb-3 whitespace-pre-wrap";
  msg.textContent = message;
  const retry = document.createElement("button");
  retry.dataset.retry = "";
  retry.className = "rounded px-3 py-1 font-semibold text-neutral-900";
  retry.style.background = "var(--accent)";
  retry.textContent = "Réessayer";
  retry.onclick = update;
  pop.append(msg, retry);
  el.append(pop);
}
