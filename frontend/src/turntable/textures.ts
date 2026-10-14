export type Texture = "black" | "picture" | "yolk" | "split" | "solid";

const cssUrl = (u: string) => `url("${u.replace(/["\\]/g, "\\$&")}")`;

export function discBackground(t: Texture, colors: string[], artworkUrl: string): string {
  const [a = "#ffb300", b = "#fff3d6"] = colors;
  switch (t) {
    case "picture":
      return artworkUrl ? `${cssUrl(artworkUrl)} center / cover` : discBackground("black", colors, "");
    case "yolk":
      return `radial-gradient(circle, ${a} 0 42%, ${b} 46% 100%)`;
    case "split":
      return `linear-gradient(90deg, ${a} 50%, ${b} 50%)`;
    case "solid":
      return `radial-gradient(circle, ${a}cc, ${a}88)`;
    default:
      return "#111";
  }
}
