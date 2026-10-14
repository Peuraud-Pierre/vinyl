import { expect, test } from "vitest";
import { parseRoute } from "./router";

test("route par défaut", () => { expect(parseRoute("")).toEqual({ route: "platine", param: "" }); });
test("route avec paramètre encodé", () => {
  expect(parseRoute("#/artiste/Beyonc%C3%A9")).toEqual({ route: "artiste", param: "Beyoncé" });
});
test("route inconnue → platine", () => { expect(parseRoute("#/nimporte")).toEqual({ route: "platine", param: "" }); });
