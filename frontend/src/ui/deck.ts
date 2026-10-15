// Touches de platine (façon touche Start/Stop Technics, noir mat) : titre précédent,
// lecture aléatoire avec témoin rouge, titre suivant.
import { icons } from "./icons";

export interface DeckDeps {
  previous(): Promise<unknown>;
  next(): Promise<unknown>;
  setShuffle(on: boolean): Promise<unknown>;
  modes(): Promise<{ shuffle: boolean; repeat: string }>;
}

export interface Deck { refresh(): Promise<void>; setEnabled(on: boolean): void; }

export function mountDeck(root: HTMLElement, deps: DeckDeps): Deck {
  root.innerHTML = `
    <div class="vy-deck">
      <button class="vy-key" data-deck="prev" aria-label="Morceau précédent">${icons.prev}</button>
      <button class="vy-key" data-deck="shuffle" aria-label="Lecture aléatoire" aria-pressed="false"><i class="vy-led"></i>${icons.shuffle}</button>
      <button class="vy-key" data-deck="next" aria-label="Morceau suivant">${icons.next}</button>
    </div>`;
  const key = (k: string) => root.querySelector<HTMLButtonElement>(`[data-deck=${k}]`)!;
  const shuffle = key("shuffle");
  let on = false;
  const paint = (v: boolean) => {
    on = v;
    shuffle.setAttribute("aria-pressed", String(v));
  };
  const refresh = () => deps.modes().then((m) => paint(m.shuffle)).catch(() => {});
  key("prev").onclick = () => { deps.previous().catch(() => {}); };
  key("next").onclick = () => { deps.next().catch(() => {}); };
  shuffle.onclick = () => {
    const want = !on;
    paint(want); // le témoin répond tout de suite ; le lecteur confirme ensuite
    deps.setShuffle(want).catch(() => {}).then(refresh);
  };
  return {
    refresh,
    setEnabled: (v) => root.querySelectorAll("button").forEach((b) => (b.disabled = !v)),
  };
}
