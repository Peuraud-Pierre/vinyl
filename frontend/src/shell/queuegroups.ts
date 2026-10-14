// File d'attente (vidéo MD Vinyl) : le nom de l'album n'est affiché qu'en tête de chaque groupe
// de titres du même album, et jamais pour l'album déjà en cours.
import type { Track } from "../api";

export function albumHeaders(currentAlbum: string, upcoming: Track[]): Array<{ track: Track; header: string }> {
  let previous = currentAlbum;
  return upcoming.map((track) => {
    const header = track.album && track.album !== previous ? track.album : "";
    if (track.album) previous = track.album;
    return { track, header };
  });
}
