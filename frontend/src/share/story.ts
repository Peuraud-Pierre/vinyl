// Image story façon MD Vinyl (cf. screen/visible vinyle (preview).png) : pochette inclinée en haut,
// vinyle avec étiquette en bas à droite, « VINYL » discret en bas à gauche.
import type { Settings } from "../api";
import { labelColorFor, labelInk } from "../turntable/label";
import { fitText, STORY_H, STORY_W } from "./layout";
import type { ShareInput } from "./sharecard";

export function storyLayout() {
  return {
    cover: { x: 40, y: 150, size: 860, angle: -8 },
    disc: { cx: 760, cy: 1490, r: 540 },
  };
}

function loadImage(src: string): Promise<HTMLImageElement | null> {
  if (!src) return Promise.resolve(null);
  return new Promise((resolve) => {
    const img = new Image();
    img.crossOrigin = "anonymous";
    img.onload = () => resolve(img);
    img.onerror = () => resolve(null);
    img.src = src;
  });
}

export async function renderStory(input: ShareInput, s: Settings, tint: string): Promise<HTMLCanvasElement> {
  const c = document.createElement("canvas");
  c.width = STORY_W;
  c.height = STORY_H;
  const g = c.getContext("2d")!;
  const { cover, disc } = storyLayout();
  const art = await loadImage(input.artworkUrl);

  // Fond teinté par la pochette, plus sombre vers le bas.
  const bg = g.createLinearGradient(0, 0, 0, STORY_H);
  bg.addColorStop(0, tint);
  bg.addColorStop(1, "#1a1210");
  g.fillStyle = bg;
  g.fillRect(0, 0, STORY_W, STORY_H);

  // Pochette inclinée avec ombre portée.
  g.save();
  g.translate(cover.x + cover.size / 2, cover.y + cover.size / 2);
  g.rotate((cover.angle * Math.PI) / 180);
  g.shadowColor = "rgba(0,0,0,.45)";
  g.shadowBlur = 60;
  g.shadowOffsetY = 30;
  if (art) g.drawImage(art, -cover.size / 2, -cover.size / 2, cover.size, cover.size);
  else { g.fillStyle = "#2a2220"; g.fillRect(-cover.size / 2, -cover.size / 2, cover.size, cover.size); }
  g.restore();

  // Vinyle noir avec sillons et reflet.
  g.save();
  g.shadowColor = "rgba(0,0,0,.5)";
  g.shadowBlur = 50;
  g.beginPath(); g.arc(disc.cx, disc.cy, disc.r, 0, Math.PI * 2);
  g.fillStyle = "#121010"; g.fill();
  g.restore();
  g.strokeStyle = "rgba(255,255,255,.045)";
  g.lineWidth = 1;
  for (let rr = disc.r * 0.5; rr < disc.r; rr += 5) { g.beginPath(); g.arc(disc.cx, disc.cy, rr, 0, Math.PI * 2); g.stroke(); }
  const shine = g.createLinearGradient(disc.cx - disc.r, disc.cy - disc.r, disc.cx + disc.r, disc.cy + disc.r);
  shine.addColorStop(0.35, "rgba(255,255,255,0)");
  shine.addColorStop(0.5, "rgba(255,255,255,.06)");
  shine.addColorStop(0.65, "rgba(255,255,255,0)");
  g.beginPath(); g.arc(disc.cx, disc.cy, disc.r, 0, Math.PI * 2);
  g.fillStyle = shine; g.fill();

  // Étiquette : noire, ou teintée en picture disc.
  const lr = disc.r * 0.46;
  const labelColor = labelColorFor(s.texture, s.texture === "picture" ? tint : null);
  const ink = labelInk(labelColor);
  g.beginPath(); g.arc(disc.cx, disc.cy, lr, 0, Math.PI * 2);
  g.fillStyle = labelColor; g.fill();
  g.strokeStyle = ink; g.lineWidth = 3;
  g.beginPath(); g.arc(disc.cx, disc.cy, lr * 0.92, 0, Math.PI * 2); g.stroke();
  g.save();
  g.translate(disc.cx, disc.cy);
  g.rotate(-0.5);
  g.fillStyle = ink;
  g.textAlign = "center";
  g.font = "800 46px -apple-system, sans-serif";
  g.fillText(fitText((t) => g.measureText(t).width, input.title, lr * 1.5), 0, -20);
  g.font = "500 30px -apple-system, sans-serif";
  g.fillText(fitText((t) => g.measureText(t).width, input.subtitle, lr * 1.4), 0, 40);
  g.restore();
  g.beginPath(); g.arc(disc.cx, disc.cy, 12, 0, Math.PI * 2);
  g.fillStyle = "#cfcac4"; g.fill();

  // Mention discrète.
  g.fillStyle = "rgba(255,255,255,.55)";
  g.font = "600 34px -apple-system, sans-serif";
  g.textAlign = "left";
  g.fillText("VINYL", 50, STORY_H - 60);
  return c;
}
