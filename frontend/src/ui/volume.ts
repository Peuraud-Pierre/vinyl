// Volume de l'ordinateur (sortie son du Mac ou du PC) : molette rotative en haut à gauche du socle de la
// platine, sans bouton. Elle tourne de -135° (0) à +135° (100) ; un arc à la couleur d'accent montre le
// niveau. On la tourne en glissant autour de son centre, à la molette de la souris (pas de 5) ou au
// clavier. Relue quand la fenêtre revient au premier plan ou que la souris arrive dessus (les touches du
// clavier ont pu changer le volume).

export interface VolumeDeps {
  get(): Promise<number>;
  set(v: number): Promise<unknown>;
}

// Envoie au système une valeur à la fois, sans file : pendant un envoi, seule la dernière valeur demandée
// attend son tour (un geste rapide ne lance pas un osascript par pixel).
export function latestOnly(send: (v: number) => Promise<unknown>): (v: number) => void {
  let busy = false;
  let pending: number | null = null;
  const run = (v: number) => {
    busy = true;
    send(v).catch(() => {}).finally(() => {
      busy = false;
      if (pending !== null) { const p = pending; pending = null; run(p); }
    });
  };
  return (v) => { if (busy) pending = v; else run(v); };
}

const clamp = (v: number) => Math.min(100, Math.max(0, v));

export const stepVolume = (v: number, deltaY: number) => clamp(v + (deltaY < 0 ? 5 : -5));

/** Angle du repère (degrés, 0 = en haut) pour un volume : -135° à 0, +135° à 100. */
export const knobAngle = (v: number) => -135 + clamp(v) * 2.7;

/** Écart d'angle le plus court de a à b (degrés, entre -180 et 180). */
export const angleDelta = (a: number, b: number) => ((b - a + 540) % 360) - 180;

/** Volume après avoir tourné la molette de l'angle « from » à l'angle « to » (sans sauter d'un bout à l'autre). */
export const turnVolume = (v: number, from: number, to: number) => clamp(v + angleDelta(from, to) / 2.7);

export interface VolumeKnob { refresh(): void; setScale(k: number): void; destroy(): void; }

export function mountVolume(root: HTMLElement, deps: VolumeDeps): VolumeKnob {
  root.innerHTML = `
    <div class="vy-knob" role="slider" tabindex="0" aria-label="Volume de l’ordinateur" aria-valuemin="0" aria-valuemax="100">
      <i class="vy-knob-ticks"></i><i class="vy-knob-ring"></i><i class="vy-knob-cap"></i>
      <span class="vy-knob-val"></span>
    </div>`;
  const knob = root.querySelector<HTMLElement>(".vy-knob")!;
  const cap = root.querySelector<HTMLElement>(".vy-knob-cap")!;
  const ring = root.querySelector<HTMLElement>(".vy-knob-ring")!;
  const val = root.querySelector<HTMLElement>(".vy-knob-val")!;
  let level = 50; // volume exact (réel pendant un glissé, pour tourner sans à-coups)
  let ok = true;
  const send = latestOnly((v) => deps.set(v));
  const paint = (v: number, available = true) => {
    level = v;
    ok = available;
    const r = Math.round(v);
    cap.style.transform = `rotate(${knobAngle(v)}deg)`;
    ring.style.setProperty("--arc", `${available ? clamp(v) * 2.7 : 0}deg`);
    val.textContent = available ? String(r) : "--";
    knob.setAttribute("aria-valuenow", String(r));
    knob.setAttribute("aria-disabled", String(!available));
    knob.title = available ? `Volume ${r} %` : "Volume indisponible";
  };
  let readAt = -Infinity;
  const refresh = () => {
    const now = performance.now();
    if (dragging || now - readAt < 2000) return;
    readAt = now;
    deps.get().then((v) => { if (!dragging) paint(v); }).catch(() => paint(level, false));
  };
  const set = (v: number) => {
    if (!ok) return;
    const before = Math.round(level);
    paint(v);
    if (Math.round(v) !== before) send(Math.round(v));
  };
  // Glisser : la molette suit l'angle du pointeur autour de son centre.
  let dragging = false;
  let lastAngle = 0;
  const pointerAngle = (e: PointerEvent) => {
    const r = knob.getBoundingClientRect();
    return (Math.atan2(e.clientX - (r.left + r.width / 2), -(e.clientY - (r.top + r.height / 2))) * 180) / Math.PI;
  };
  knob.addEventListener("pointerdown", (e) => {
    if (!ok) return;
    knob.setPointerCapture(e.pointerId);
    dragging = true;
    lastAngle = pointerAngle(e);
  });
  knob.addEventListener("pointermove", (e) => {
    if (!dragging) return;
    const a = pointerAngle(e);
    set(turnVolume(level, lastAngle, a));
    lastAngle = a;
  });
  const end = () => { dragging = false; };
  knob.addEventListener("pointerup", end);
  knob.addEventListener("pointercancel", end);
  knob.addEventListener("keydown", (e) => {
    const next = ({ ArrowRight: level + 5, ArrowUp: level + 5, ArrowLeft: level - 5, ArrowDown: level - 5, Home: 0, End: 100 } as Record<string, number>)[e.key];
    if (next === undefined) return;
    e.preventDefault();
    set(clamp(Math.round(next)));
  });
  knob.addEventListener("wheel", (e) => {
    if (!ok || e.deltaY === 0) return;
    e.preventDefault();
    set(stepVolume(Math.round(level), e.deltaY));
  }, { passive: false });
  knob.addEventListener("pointerenter", refresh);
  window.addEventListener("focus", refresh);
  refresh();
  return {
    refresh,
    // zoom comme l'égaliseur : dessinée à sa taille d'affichage, donc nette.
    setScale(k) { knob.style.zoom = String(k); },
    destroy() { window.removeEventListener("focus", refresh); },
  };
}
