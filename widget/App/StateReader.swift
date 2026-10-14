import Foundation

/// Construit le nouvel état à partir de Musique. Appelé hors du fil principal.
enum StateReader {
    /// nil = rien de fiable (erreur passagère) : garder l'état précédent.
    static func read(previous: NowPlaying, bridge: MusicBridge, store: StateStore, now: Date) -> NowPlaying? {
        let accent = AccentReader.read()
        switch bridge.readState() {
        case .failure(.permissionDenied):
            var n = previous
            n.availability = .permissionDenied
            n.accentHex = accent
            return n
        case .failure(let e):
            NSLog("Vinyl Widget : lecture de Musique impossible : %@", "\(e)")
            return nil
        case .success(.closed):
            return previous.closed(accentHex: accent, now: now)
        case .success(.stopped):
            guard previous.trackID != nil else { return .neverPlayed(accentHex: accent, now: now) }
            var n = previous
            n.availability = .running
            n.state = .stopped
            n.position = 0
            n.capturedAt = now
            n.accentHex = accent
            return n
        case .success(.track(let t)):
            var n = NowPlaying(availability: .running, state: t.state, trackID: t.trackID, title: t.title,
                               artist: t.artist, album: t.album, duration: t.duration, position: t.position,
                               capturedAt: now, artworkFile: nil, accentHex: accent)
            if previous.trackID == t.trackID { n.artworkFile = previous.artworkFile }
            if n.artworkFile == nil { n.artworkFile = fetchArtwork(trackID: t.trackID, bridge: bridge, store: store) }
            return n
        }
    }

    private static func fetchArtwork(trackID: String, bridge: MusicBridge, store: StateStore) -> String? {
        let raw = FileManager.default.temporaryDirectory.appendingPathComponent("vinylwidget-artwork.raw")
        try? FileManager.default.removeItem(at: raw)
        defer { try? FileManager.default.removeItem(at: raw) }
        guard bridge.writeArtwork(to: raw, trackID: trackID), let jpeg = Artwork.jpegThumbnail(from: raw) else { return nil }
        return try? store.saveArtwork(jpeg, trackID: trackID)
    }
}
