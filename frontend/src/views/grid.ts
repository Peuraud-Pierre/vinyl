import type { Item } from "../api";

export function escapeHtml(s: string): string {
  return s.replace(/[&<>"']/g, (c) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" })[c]!);
}

export function initials(title: string): string {
  const parts = title.trim().split(/\s+/).filter(Boolean);
  if (!parts.length) return "♪";
  return parts.slice(0, 2).map((p) => p[0]!.toUpperCase()).join("");
}

export function itemCard(item: Item): string {
  const art = item.artworkUrl
    ? `<img src="${escapeHtml(item.artworkUrl)}" class="aspect-square w-full rounded-md object-cover" loading="lazy" alt="">`
    : `<div class="flex aspect-square w-full items-center justify-center rounded-md bg-neutral-800 text-2xl text-neutral-500" data-art>${escapeHtml(initials(item.title))}</div>`;
  return `<button class="group flex flex-col gap-1 text-left" data-id="${escapeHtml(item.id)}">
    ${art}
    <span class="truncate text-sm font-medium">${escapeHtml(item.title)}</span>
    <span class="truncate text-xs text-neutral-500">${escapeHtml(item.subtitle)}</span>
  </button>`;
}
