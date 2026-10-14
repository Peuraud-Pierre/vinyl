import { expect, test } from "vitest";
import { escapeHtml, initials, itemCard } from "./grid";

test("échappe le HTML", () => {
  expect(escapeHtml(`<b>"Rock" & Roll</b>`)).toBe("&lt;b&gt;&quot;Rock&quot; &amp; Roll&lt;/b&gt;");
});
test("initiales", () => {
  expect(initials("daft punk")).toBe("DP");
  expect(initials("")).toBe("♪");
});
test("carte sans pochette affiche les initiales, titre échappé", () => {
  const html = itemCard({ id: "1", kind: "album", title: "<x>", subtitle: "", artworkUrl: "", url: "" });
  expect(html).toContain("&lt;x&gt;");
  expect(html).toContain('data-id="1"');
});
