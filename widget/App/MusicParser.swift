import Foundation

struct TrackInfo: Equatable {
    var state: NowPlaying.PlayerState
    var trackID: String
    var title: String
    var artist: String
    var album: String
    var duration: TimeInterval
    var position: TimeInterval
}

enum MusicReading: Equatable {
    case closed
    case stopped
    case track(TrackInfo)
}

/// Analyse la sortie du script d'état de MusicBridge (champs séparés par ASCII 31).
enum MusicParser {
    static let separator = "\u{1F}"

    static func parse(_ output: String) -> MusicReading? {
        let text = output.hasSuffix("\n") ? String(output.dropLast()) : output
        if text == "closed" { return .closed }
        if text == "stopped" { return .stopped }
        let f = text.components(separatedBy: separator)
        guard f.count == 7, let state = NowPlaying.PlayerState(rawValue: f[0]) else { return nil }
        return .track(TrackInfo(state: state, trackID: f[1], title: f[2], artist: f[3], album: f[4],
                                duration: seconds(f[5]), position: seconds(f[6])))
    }

    /// Secondes AppleScript ("213,5" ou "213.5"). Valeur absente ou invalide → 0.
    static func seconds(_ s: String) -> TimeInterval {
        let v = Double(s.trimmingCharacters(in: .whitespaces).replacingOccurrences(of: ",", with: ".")) ?? 0
        return v.isFinite && v > 0 ? v : 0
    }
}
