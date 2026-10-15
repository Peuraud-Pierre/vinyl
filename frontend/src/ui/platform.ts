// Système de l'app (« darwin » | « windows »), lu une fois : sert à masquer ce qui n'existe pas.
import { api } from "../api";

let cached: Promise<string> | null = null;
export const platform = (): Promise<string> => (cached ??= api.Platform().catch(() => "darwin"));
