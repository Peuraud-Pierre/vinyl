import Foundation
import WidgetKit

extension NowPlaying {
    /// Aperçu de la galerie de widgets.
    static let sample = NowPlaying(availability: .running, state: .playing, trackID: "sample",
                                   title: "Titre du morceau", artist: "Artiste", album: "Album",
                                   duration: 240, position: 80, capturedAt: Date(), artworkFile: nil,
                                   accentHex: defaultAccent)
}

struct Provider: TimelineProvider {
    private func load(now: Date) -> NowPlaying {
        guard let dir = AppGroup.containerURL, let np = StateStore(directory: dir).load() else {
            return .neverPlayed(now: now)
        }
        return np
    }

    func placeholder(in context: Context) -> VinylEntry {
        VinylEntry(date: Date(), nowPlaying: .sample, position: 80)
    }

    func getSnapshot(in context: Context, completion: @escaping (VinylEntry) -> Void) {
        let now = Date()
        let np = context.isPreview ? .sample : load(now: now)
        completion(TimelineBuilder.entries(for: np, now: now).entries[0])
    }

    func getTimeline(in context: Context, completion: @escaping (Timeline<VinylEntry>) -> Void) {
        let now = Date()
        let r = TimelineBuilder.entries(for: load(now: now), now: now)
        completion(Timeline(entries: r.entries, policy: r.refreshAtEnd ? .atEnd : .never))
    }
}
