// Mémoire de l'étagère pendant la session : au retour d'une page album, on retrouve
// les éléments déjà chargés et la sélection, sans relire toute la bibliothèque.
import type { Item } from "../api";

export interface ShelfState { items: Item[]; next: number; focus: number; }

const memory = new Map<string, ShelfState>();

export function remember(key: string, s: ShelfState): void {
  memory.set(key, { items: [...s.items], next: s.next, focus: s.focus });
}

export function recall(key: string): ShelfState | undefined {
  const s = memory.get(key);
  return s && { items: [...s.items], next: s.next, focus: s.focus };
}
