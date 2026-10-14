// Étiquette imprimée du disque : titre, artiste, mentions en cercle, code-barres.
import { escapeHtml } from "../views/grid";
import { hexToRgb, luminance } from "../ui/theme";

export function labelInk(color: string): string {
  const rgb = hexToRgb(color);
  return rgb && luminance(rgb) > 0.3 ? "#1d1714" : "#f2ede8";
}

const clip = (s: string, n: number) => (s.length > n ? s.slice(0, n - 1).trimEnd() + "…" : s);

let seq = 0;

export function labelMarkup(o: { title: string; artist: string; color: string; ring: string }): string {
  const ink = labelInk(o.color);
  const id = `vy-ring-${++seq}`;
  return `
  <div class="absolute inset-0 rounded-full" style="background:${o.color}"></div>
  <div class="absolute rounded-full" style="inset:4%;border:1.5px solid ${ink};opacity:.85"></div>
  <svg class="absolute inset-0 h-full w-full" viewBox="0 0 200 200" aria-hidden="true">
    <defs><path id="${id}" d="M100,100 m-80,0 a80,80 0 1,1 160,0 a80,80 0 1,1 -160,0"></path></defs>
    <text font-size="6.6" letter-spacing="1.1" fill="${ink}" font-family="-apple-system, system-ui, sans-serif"><textPath href="#${id}">${escapeHtml(o.ring)}</textPath></text>
  </svg>
  <div class="absolute flex flex-col" style="left:15%;right:12%;top:32%;gap:.35em;color:${ink}">
    <span style="font-size:11.5cqw;font-weight:800;line-height:1.05">${escapeHtml(clip(o.title, 28))}</span>
    <span style="font-size:6.5cqw;font-weight:500;opacity:.85">${escapeHtml(clip(o.artist, 34))}</span>
  </div>
  <div class="absolute" style="left:22%;bottom:17%;width:13%;height:6%;opacity:.8;background:repeating-linear-gradient(90deg,${ink} 0 1px,transparent 1px 2px,${ink} 2px 4px,transparent 4px 5px)"></div>
  <div class="absolute rounded-full" style="left:47%;top:47%;width:6%;height:6%;background:radial-gradient(circle,#d9d4ce 0 35%,#6b6560 40%,#2a2726 100%)"></div>`;
}
