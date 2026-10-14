import Foundation

/// Décide si un nouvel état mérite un rafraîchissement du widget (budget de rechargements limité).
enum StateDiff {
    static let driftTolerance: TimeInterval = 3

    static func shouldPublish(old: NowPlaying, new: NowPlaying) -> Bool {
        var a = old, b = new
        a.position = 0; a.capturedAt = .distantPast
        b.position = 0; b.capturedAt = .distantPast
        if a != b { return true }
        return abs(old.position(at: new.capturedAt) - new.position) > driftTolerance
    }
}
