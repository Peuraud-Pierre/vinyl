import type { Settings, Snapshot } from "../api";
import { api } from "../api";
import { ARM_GEOMETRY, ARM_LIFT, ARM_REST, CLICK_TOLERANCE, angleForProgress, armRelease, dragAngleFromPointer } from "./arm";
import { estimatePosition } from "./clock";
import { playCrackle } from "./crackle";
import { Spinner } from "./spin";
import { discBackground, type Texture } from "./textures";

type Variant = "full" | "widget";

// Le bras (8px de large) tourne autour de transform-origin 4px 8px, placé sur ARM_GEOMETRY.pivot.
// Sa tête de lecture dépasse de 12px sous le bras : longueur visuelle = hauteur + 12 - 8.
const ARM_LEFT = ARM_GEOMETRY.pivot.x - 4;
const ARM_TOP = ARM_GEOMETRY.pivot.y - 8;
const ARM_HEIGHT = ARM_GEOMETRY.length - 4;

const TEMPLATE_FULL = `
  <div class="tt-plinth relative rounded-2xl" style="width:420px;height:320px;background:linear-gradient(135deg,#2b2b2b,#1a1a1a);box-shadow:0 20px 50px rgba(0,0,0,.6),inset 0 1px 0 rgba(255,255,255,.08)">
    <div class="absolute rounded-full flex items-center justify-center" style="left:20px;top:18px;width:284px;height:284px;background:#333;box-shadow:inset 0 0 0 6px #555">
      <div class="tt-disc relative rounded-full flex items-center justify-center" style="width:264px;height:264px"></div>
    </div>
    <div class="tt-arm absolute cursor-grab" style="left:${ARM_LEFT}px;top:${ARM_TOP}px;width:8px;height:${ARM_HEIGHT}px;transform-origin:4px 8px;border-radius:4px;background:linear-gradient(#eee,#9a9a9a)">
      <div class="absolute rounded-full" style="top:-14px;left:-14px;width:36px;height:36px;background:#aaa;box-shadow:inset 0 0 0 5px #777"></div>
      <div class="absolute" style="bottom:-12px;left:-7px;width:22px;height:30px;border-radius:4px;background:var(--accent)"></div>
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

  constructor(root: HTMLElement, opts: { variant: Variant }) {
    root.innerHTML = opts.variant === "full" ? TEMPLATE_FULL : TEMPLATE_WIDGET;
    this.disc = root.querySelector(".tt-disc")!;
    this.disc.innerHTML = DISC_INNER;
    this.label = root.querySelector(".tt-label")!;
    this.arm = root.querySelector(".tt-arm");
    this.label.addEventListener("click", () => api.PlayPause());
    if (this.arm) this.bindArm(this.arm);
    this.paint();
    this.raf = requestAnimationFrame(this.frame);
  }

  update(s: Snapshot): void {
    const prevTrack = this.snap?.nowPlaying.trackId;
    const prevArt = this.snap?.nowPlaying.artworkUrl;
    this.snap = s;
    this.receivedAt = performance.now();
    if (s.nowPlaying.trackId !== prevTrack || s.nowPlaying.artworkUrl !== prevArt) this.paint();
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
    const pivot = () => {
      const plinth = arm.parentElement!.getBoundingClientRect();
      return { x: plinth.left + ARM_GEOMETRY.pivot.x, y: plinth.top + ARM_GEOMETRY.pivot.y };
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
