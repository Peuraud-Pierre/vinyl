import { expect, test } from "vitest";
import { STORY_H, STORY_W } from "./layout";
import { storyLayout } from "./story";

test("format story 1080×1920", () => {
  expect([STORY_W, STORY_H]).toEqual([1080, 1920]);
});
test("pochette en haut, inclinée ; vinyle en bas à droite, rogné par le bord", () => {
  const { cover, disc } = storyLayout();
  expect(cover.y).toBeLessThan(STORY_H / 3);
  expect(cover.angle).toBeLessThan(0);
  expect(disc.cy).toBeGreaterThan(STORY_H * 0.6);
  expect(disc.cx + disc.r).toBeGreaterThan(STORY_W);
  expect(disc.cy + disc.r).toBeGreaterThan(STORY_H * 0.9);
});
