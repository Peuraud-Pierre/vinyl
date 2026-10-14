// Icônes redessinées (SVG en trait, couleur = currentColor).
const svg = (size: number, body: string, fill = false) =>
  `<svg width="${size}" height="${size}" viewBox="0 0 24 24" ${fill ? "" : 'fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"'} aria-hidden="true">${body}</svg>`;

export const icons = {
  share: svg(20, '<rect x="3" y="6" width="9" height="15" rx="3"/><path d="M14 4a7 7 0 0 1 7 7"/><path d="M18 4h3v3"/>'),
  settings: svg(20, '<path d="M12 2.5l8.2 4.75v9.5L12 21.5l-8.2-4.75v-9.5z"/><circle cx="12" cy="12" r="3.4"/>'),
  library: svg(24, '<path d="M4 4v16M8 4v16"/><path d="M12 4l7 2v12l-7 2z" fill="currentColor"/>'),
  play: svg(20, '<path d="M7 4.5v15l12.5-7.5z" fill="currentColor"/>', true),
  pause: svg(20, '<rect x="5" y="4" width="5" height="16" rx="1.5" fill="currentColor"/><rect x="14" y="4" width="5" height="16" rx="1.5" fill="currentColor"/>', true),
  queue: svg(22, '<path d="M4 6h14M4 11h9M4 16h5"/><path d="M14 17.5a3.5 3.5 0 1 0 1-4.8"/><path d="M14.6 10.6l.5 2.3-2.3.4"/>'),
  more: svg(20, '<circle cx="5" cy="12" r="2" fill="currentColor"/><circle cx="12" cy="12" r="2" fill="currentColor"/><circle cx="19" cy="12" r="2" fill="currentColor"/>', true),
  shelf: svg(20, '<path d="M4 5v14M9 5v14M14 5v14M19 7v12"/>'),
  stack: svg(20, '<path d="M4 6h16M4 10h16M4 14h16M4 18h16"/>'),
  search: svg(20, '<circle cx="10.5" cy="10.5" r="6.5"/><path d="M15.5 15.5L21 21"/>'),
  back: svg(20, '<path d="M15 4l-8 8 8 8"/>'),
  close: svg(20, '<path d="M6 6l12 12M18 6L6 18"/>'),
  prev: svg(22, '<path d="M6 5v14"/><path d="M19 5L9 12l10 7z" fill="currentColor"/>'),
  next: svg(22, '<path d="M18 5v14"/><path d="M5 5l10 7-10 7z" fill="currentColor"/>'),
  palette: svg(20, '<path d="M12 3a9 9 0 1 0 0 18c1.2 0 1.6-1 1.1-1.9-.6-1.1.1-2.1 1.3-2.1H17a4 4 0 0 0 4-4c0-5.5-4-10-9-10z"/><circle cx="7.5" cy="11" r="1.2"/><circle cx="10.5" cy="7.5" r="1.2"/><circle cx="15" cy="7.5" r="1.2"/>'),
  person: svg(20, '<circle cx="12" cy="8" r="4"/><path d="M4 21c1.5-4 4.5-6 8-6s6.5 2 8 6"/>'),
  shuffle: svg(22, '<path d="M3 7h3.5c2.6 0 4 1.2 5.5 5s2.9 5 5.5 5H21"/><path d="M3 17h3.5c1.4 0 2.5-.4 3.4-1.3"/><path d="M14.1 8.3C15 7.4 16.1 7 17.5 7H21"/><path d="M18 4l3 3-3 3"/><path d="M18 14l3 3-3 3"/>'),
  repeat: svg(22, '<path d="M4 11V9a3 3 0 0 1 3-3h13"/><path d="M17 3l3 3-3 3"/><path d="M20 13v2a3 3 0 0 1-3 3H4"/><path d="M7 21l-3-3 3-3"/>'),
  repeatOne: svg(22, '<path d="M4 11V9a3 3 0 0 1 3-3h13"/><path d="M17 3l3 3-3 3"/><path d="M20 13v2a3 3 0 0 1-3 3H4"/><path d="M7 21l-3-3 3-3"/><path d="M11.5 10.5l1.5-1v5"/>'),
  mini: svg(20, '<circle cx="12" cy="12" r="8"/><circle cx="12" cy="12" r="2.5"/>'),
};
