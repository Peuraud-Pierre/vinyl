import type { Settings, Snapshot } from "../api";
import { api } from "../api";
import { ARM_GEOMETRY, ARM_LIFT, ARM_REST, CLICK_TOLERANCE, angleForProgress, armRelease, dragAngleFromPointer } from "./arm";
import { estimatePosition } from "./clock";
import { playCrackle } from "./crackle";
import { Spinner } from "./spin";
import { discBackground, type Texture } from "./textures";
import { labelColorFor, labelMarkup } from "./label";
import { imageTint } from "../ui/theme";

type Variant = "stage" | "widget";

// Scène de la platine (960×660) : pochette, disque, bras. Coordonnées = ARM_GEOMETRY.
export const STAGE_W = 960;
export const STAGE_H = 660;
const G = ARM_GEOMETRY;
const DISC_LEFT = G.discCenter.x - G.discRadius;
const DISC_TOP = G.discCenter.y - G.discRadius;
const DISC_SIZE = G.discRadius * 2;
const ARM_BOX = { ox: 38, oy: 40 }; // position du pivot dans la boîte du bras

// Noir fumé marbré : base translucide, marbrures, sillons.
const SMOKE = [
  "repeating-radial-gradient(circle at 50% 50%, rgba(255,255,255,.035) 0 1px, transparent 1px 3px)",
  "radial-gradient(ellipse 30% 12% at 30% 28%, rgba(0,0,0,.45), transparent 70%)",
  "radial-gradient(ellipse 22% 9% at 72% 66%, rgba(0,0,0,.4), transparent 70%)",
  "radial-gradient(ellipse 18% 30% at 22% 70%, rgba(0,0,0,.35), transparent 70%)",
  "radial-gradient(ellipse 26% 8% at 66% 22%, rgba(255,255,255,.05), transparent 70%)",
  "radial-gradient(circle at 50% 50%, rgba(40,36,35,.93) 0 46%, rgba(26,23,22,.95) 70%, rgba(12,10,10,.97) 100%)",
].join(",");

const TEMPLATE_STAGE = `
  <div class="tt-stage relative" style="width:${STAGE_W}px;height:${STAGE_H}px">
    <div class="tt-sleeve absolute overflow-hidden bg-cover bg-center" style="left:20px;top:60px;width:520px;height:520px;transform:rotate(-2.2deg);border-radius:10px;border:6px solid #0c0909;background-color:#15100f;box-shadow:0 30px 60px rgba(20,8,6,.55),0 4px 10px rgba(0,0,0,.4)">
      <div class="tt-sleeve-empty absolute inset-0 flex items-end p-7 text-[26px] font-semibold" style="color:#e9d6c9;background:radial-gradient(circle at 70% 62%, var(--accent) 0 18%, color-mix(in srgb, var(--accent) 50%, #15100f) 36%, #15100f 60%)"></div>
    </div>
    <div class="absolute rounded-full" style="left:${DISC_LEFT}px;top:${DISC_TOP}px;width:${DISC_SIZE}px;height:${DISC_SIZE}px;box-shadow:0 26px 50px rgba(20,8,6,.55)">
      <div class="tt-disc absolute inset-0 overflow-hidden rounded-full">
        <div class="tt-base absolute inset-0 rounded-full"></div>
        <div class="absolute rounded-full pointer-events-none" style="inset:1px;box-shadow:inset 0 0 0 10px rgba(255,255,255,.03),inset 0 0 0 54px rgba(0,0,0,.12)"></div>
        <button class="tt-label no-drag absolute rounded-full" style="left:27.5%;top:27.5%;width:45%;height:45%;container-type:inline-size" aria-label="Lecture / pause"></button>
      </div>
      <div class="absolute inset-0 rounded-full pointer-events-none" style="background:conic-gradient(from 20deg, transparent 0deg, rgba(255,255,255,.07) 24deg, transparent 50deg, transparent 180deg, rgba(255,255,255,.05) 206deg, transparent 236deg)"></div>
    </div>
    <div class="glass absolute pointer-events-none" style="left:${G.pivot.x - 48}px;top:${G.pivot.y - 48}px;width:96px;height:200px;border-radius:48px;background:rgba(255,236,230,.16)"></div>
    <div class="tt-arm absolute cursor-grab" style="left:${G.pivot.x - ARM_BOX.ox}px;top:${G.pivot.y - ARM_BOX.oy}px;width:120px;height:${ARM_BOX.oy + G.length + 20}px;transform-origin:${ARM_BOX.ox}px ${ARM_BOX.oy}px">
      <svg width="120" height="${ARM_BOX.oy + G.length + 20}" style="overflow:visible" aria-hidden="true">
        <defs><linearGradient id="tt-chrome" x1="0" x2="1"><stop offset="0" stop-color="#6d6a68"/><stop offset=".45" stop-color="#e6e3e0"/><stop offset="1" stop-color="#7c7876"/></linearGradient></defs>
        <rect x="20" y="0" width="34" height="12" rx="3" fill="#141212"/>
        <rect x="9" y="12" width="58" height="40" rx="6" fill="#1b1918"/>
        <rect x="30" y="52" width="16" height="44" rx="3" fill="#1b1918"/>
        <rect x="33" y="96" width="10" height="${G.length - 110}" fill="url(#tt-chrome)"/>
        <rect x="30" y="${G.length - 18}" width="16" height="30" rx="3" fill="#1b1918"/>
        <g transform="rotate(-28 38 ${G.length + 10})"><rect x="22" y="${G.length + 4}" width="40" height="62" rx="6" fill="#1f1d1c"/><rect x="54" y="${G.length + 50}" width="22" height="5" rx="2" fill="#2b2928"/></g>
      </svg>
    </div>
  </div>`;

const TEMPLATE_WIDGET = `<div class="tt-disc relative rounded-full flex items-center justify-center" style="width:200px;height:200px;box-shadow:0 12px 30px rgba(0,0,0,.55)"></div>`;

const DISC_INNER = `
  <div class="absolute inset-0 rounded-full pointer-events-none" style="background:repeating-radial-gradient(circle,rgba(0,0,0,.25) 0 1px,transparent 1px 3px)"></div>
  <div class="absolute inset-0 rounded-full pointer-events-none" style="background:conic-gradient(transparent 0 20%,rgba(255,255,255,.14) 25%,transparent 30% 70%,rgba(255,255,255,.14) 75%,transparent 80%)"></div>
  <button class="tt-label no-drag relative z-10 rounded-full bg-center bg-cover" style="width:36%;height:36%" aria-label="Lecture / pause"></button>`;

const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;

export class Turntable {
  private disc: HTMLElement;
  private label: HTMLElement;
  private arm: HTMLElement | null;
  private spinner = new Spinner();
  private snap: Snapshot | null = null;
  private receivedAt = 0;
  private dragAngle: number | null = null;
  private appearance: Settings | null = null;
  private raf = 0;
  private last = performance.now();
  private armWasDown = false;

  private variant: Variant;
  private labelTint: string | null = null;
  private tintFor = "";

  constructor(root: HTMLElement, opts: { variant: Variant }) {
    this.variant = opts.variant;
    root.innerHTML = opts.variant === "stage" ? TEMPLATE_STAGE : TEMPLATE_WIDGET;
    this.disc = root.querySelector(".tt-disc")!;
    if (opts.variant === "widget") this.disc.innerHTML = DISC_INNER;
    this.label = root.querySelector(".tt-label")!;
    this.arm = root.querySelector(".tt-arm");
    this.label.addEventListener("click", () => api.PlayPause());
    if (this.arm) this.bindArm(this.arm);
    // Pochette de la platine : un clic la fait glisser puis revenir, avec un ressort.
    const sleeve = root.querySelector<HTMLElement>(".tt-sleeve");
    if (sleeve) {
      sleeve.style.cursor = "pointer";
      sleeve.addEventListener("click", () => {
        sleeve.classList.remove("vy-bounce");
        void sleeve.offsetWidth; // relance l'animation si on clique à nouveau
        sleeve.classList.add("vy-bounce");
      });
      sleeve.addEventListener("animationend", () => sleeve.classList.remove("vy-bounce"));
    }
    this.paint();
    this.raf = requestAnimationFrame(this.frame);
  }

  update(s: Snapshot): void {
    const prevTrack = this.snap?.nowPlaying.trackId;
    const prevArt = this.snap?.nowPlaying.artworkUrl;
    this.snap = s;
    this.receivedAt = performance.now();
    if (s.nowPlaying.trackId !== prevTrack || s.nowPlaying.artworkUrl !== prevArt) this.paint();
    if (this.variant === "stage" && s.nowPlaying.artworkUrl !== this.tintFor) this.loadLabelColor(s.nowPlaying.artworkUrl);
  }

  setAppearance(s: Settings): void {
    this.appearance = s;
    this.paint();
  }

  destroy(): void {
    cancelAnimationFrame(this.raf);
  }

  private get playing(): boolean {
    return !!this.snap && this.snap.problem === "" && this.snap.nowPlaying.state === "playing";
  }

  private paint(): void {
    if (this.variant === "widget") return this.paintWidget();
    const np = this.snap?.nowPlaying;
    const art = np?.artworkUrl ?? "";
    const tex = (this.appearance?.texture ?? "black") as Texture;
    const colors = this.appearance?.discColors ?? [];
    const base = this.disc.querySelector<HTMLElement>(".tt-base")!;
    base.style.background = tex === "black" ? SMOKE : discBackground(tex, colors, art);
    const sleeve = this.disc.closest(".tt-stage")!.querySelector<HTMLElement>(".tt-sleeve")!;
    sleeve.style.backgroundImage = art ? cssUrl(art) : "";
    const empty = sleeve.querySelector<HTMLElement>(".tt-sleeve-empty")!;
    empty.hidden = !!art;
    empty.textContent = np?.artist ?? "";
    const title = np?.title || "Vinyl";
    const ring = ["VINYL", "33⅓ RPM", np?.source === "spotify" ? "SPOTIFY" : "APPLE MUSIC", np?.album || "FACE A", "STÉRÉO"].join(" · ") + " ·";
    const color = labelColorFor(tex, this.labelTint);
    this.label.innerHTML = labelMarkup({ title, artist: np?.artist ?? "", color, ring });
  }

  private async loadLabelColor(art: string): Promise<void> {
    this.tintFor = art;
    const tint = await imageTint(art);
    if (this.tintFor !== art) return; // un autre morceau est arrivé entre-temps
    this.labelTint = tint;
    this.paint();
  }

  private paintWidget(): void {
    const art = this.snap?.nowPlaying.artworkUrl ?? "";
    const tex = (this.appearance?.texture ?? "black") as Texture;
    const colors = this.appearance?.discColors ?? [];
    this.disc.style.background = discBackground(tex, colors, art);
    this.label.style.backgroundImage = art ? cssUrl(art) : "";
    this.label.style.backgroundColor = art ? "" : "var(--accent)";
    this.label.style.opacity = tex === "picture" && art ? "0" : "1";
  }

  private frame = (now: number) => {
    const dt = Math.min(100, now - this.last);
    this.last = now;
    this.disc.style.transform = `rotate(${this.spinner.step(dt, this.playing)}deg)`;
    if (this.arm) {
      let angle = ARM_REST;
      if (this.dragAngle !== null) angle = this.dragAngle;
      else if (this.playing && this.snap) {
        const np = this.snap.nowPlaying;
        angle = angleForProgress(estimatePosition(np.positionMs, np.durationMs, true, this.receivedAt, now), np.durationMs);
      }
      const down = angle >= ARM_LIFT;
      if (down && !this.armWasDown && this.dragAngle === null && this.appearance?.crackle) playCrackle();
      this.armWasDown = down;
      this.arm.style.transform = `rotate(${angle}deg)`;
      this.arm.style.transition = this.dragAngle === null ? "transform .6s ease" : "none";
    }
    this.raf = requestAnimationFrame(this.frame);
  };

  private bindArm(arm: HTMLElement): void {
    // La scène peut être mise à l'échelle (transform) : on convertit ses coordonnées en pixels écran.
    const pivot = () => {
      const stage = arm.parentElement!.getBoundingClientRect();
      const k = stage.width / STAGE_W;
      return { x: stage.left + ARM_GEOMETRY.pivot.x * k, y: stage.top + ARM_GEOMETRY.pivot.y * k };
    };
    let start = { x: 0, y: 0 };
    let moved = 0;
    const end = () => {
      this.dragAngle = null;
    };
    arm.addEventListener("pointerdown", (e) => {
      const angle = dragAngleFromPointer(pivot(), { x: e.clientX, y: e.clientY });
      if (angle === null) return; // clic sur la tête du pivot
      arm.setPointerCapture(e.pointerId);
      start = { x: e.clientX, y: e.clientY };
      moved = 0;
      // Un clic simple ne doit pas faire sauter le bras : on part de sa position affichée.
      this.dragAngle = this.playing ? angle : ARM_REST;
    });
    arm.addEventListener("pointermove", (e) => {
      if (this.dragAngle === null) return;
      moved = Math.max(moved, Math.hypot(e.clientX - start.x, e.clientY - start.y));
      if (moved < CLICK_TOLERANCE) return;
      const angle = dragAngleFromPointer(pivot(), { x: e.clientX, y: e.clientY });
      if (angle !== null) this.dragAngle = angle;
    });
    arm.addEventListener("pointercancel", end);
    arm.addEventListener("pointerup", async () => {
      if (this.dragAngle === null) return;
      const action = armRelease({
        endAngle: this.dragAngle, movedPx: moved, playing: this.playing, durationMs: this.snap?.nowPlaying.durationMs ?? 0,
      });
      end();
      try {
        if (action.kind === "pause") await api.PlayPause();
        if (action.kind === "play") await api.PlayPause();
        if (action.kind === "seek") {
          if (this.appearance?.crackle) playCrackle();
          await api.SeekMs(action.positionMs);
          if (action.resume) await api.PlayPause();
        }
      } catch {
        // Aucun lecteur actif : la platine affiche déjà le message adéquat.
      }
    });
  }
}
