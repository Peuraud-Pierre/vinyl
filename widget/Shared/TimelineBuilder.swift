import Foundation

/// Une image du widget à un instant donné.
struct VinylEntry {
    let date: Date
    let nowPlaying: NowPlaying
    let position: TimeInterval

    var isPlaying: Bool { nowPlaying.availability == .running && nowPlaying.state == .playing }
    var armAngle: Double { ArmMath.angle(isPlaying: isPlaying, position: position, duration: nowPlaying.duration) }
    var progress: Double { ArmMath.progress(position: position, duration: nowPlaying.duration) }
}

enum TimelineBuilder {
    static let step: TimeInterval = 15
    static let maxEntries = 80

    /// En lecture : une entrée toutes les 15 s jusqu'à la fin du morceau (refreshAtEnd = true).
    /// Sinon : une seule entrée, et c'est l'assistant qui demandera le prochain rafraîchissement.
    static func entries(for np: NowPlaying, now: Date) -> (entries: [VinylEntry], refreshAtEnd: Bool) {
        let first = VinylEntry(date: now, nowPlaying: np, position: np.position(at: now))
        guard first.isPlaying, np.duration > 0, first.position < np.duration else { return ([first], false) }
        var list = [first]
        var t = now.addingTimeInterval(step)
        while list.count < maxEntries {
            let p = np.position(at: t)
            list.append(VinylEntry(date: t, nowPlaying: np, position: p))
            if p >= np.duration { break }
            t = t.addingTimeInterval(step)
        }
        return (list, true)
    }
}
