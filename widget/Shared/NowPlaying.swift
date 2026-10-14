import Foundation

/// État publié par l'assistant et affiché par le widget.
struct NowPlaying: Codable, Equatable {
    enum PlayerState: String, Codable { case playing, paused, stopped }
    enum Availability: String, Codable { case running, closed, permissionDenied, neverPlayed }

    var availability: Availability
    var state: PlayerState
    var trackID: String?          // identifiant persistant Musique
    var title: String?
    var artist: String?
    var album: String?
    var duration: TimeInterval    // secondes, 0 si inconnue
    var position: TimeInterval    // secondes, au moment `capturedAt`
    var capturedAt: Date
    var artworkFile: String?      // nom du fichier dans le conteneur
    var accentHex: String         // "#rrggbb"

    static let defaultAccent = "#f5c542"

    static func neverPlayed(accentHex: String = defaultAccent, now: Date = Date()) -> NowPlaying {
        NowPlaying(availability: .neverPlayed, state: .stopped, trackID: nil, title: nil, artist: nil, album: nil,
                   duration: 0, position: 0, capturedAt: now, artworkFile: nil, accentHex: accentHex)
    }

    /// Position estimée à `date` : elle n'avance qu'en lecture, bornée à [0, durée] si la durée est connue.
    func position(at date: Date) -> TimeInterval {
        var p = position
        if availability == .running && state == .playing {
            p += date.timeIntervalSince(capturedAt)
        }
        p = max(0, p)
        return duration > 0 ? min(p, duration) : p
    }

    /// État « arrêté » construit sans interroger Musique (notification « Stopped »).
    func stopped(accentHex: String, now: Date) -> NowPlaying {
        guard trackID != nil else { return .neverPlayed(accentHex: accentHex, now: now) }
        var n = self
        n.availability = .running
        n.state = .stopped
        n.position = 0
        n.capturedAt = now
        n.accentHex = accentHex
        return n
    }

    /// État « Musique fermée » : garde le dernier morceau, ou « jamais rien joué » s'il n'y en a pas.
    func closed(accentHex: String, now: Date) -> NowPlaying {
        guard trackID != nil else { return .neverPlayed(accentHex: accentHex, now: now) }
        var n = self
        n.availability = .closed
        n.state = .paused
        n.position = position(at: now)
        n.capturedAt = now
        n.accentHex = accentHex
        return n
    }
}
