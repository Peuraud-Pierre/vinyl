import Foundation

/// Angles du bras, identiques à frontend/src/turntable/arm.ts.
enum ArmMath {
    static let rest = 4.0   // bras levé
    static let start = 18.0 // aiguille au bord du disque (début du morceau)
    static let end = 34.0   // aiguille près de l'étiquette (fin du morceau)

    static func progress(position: TimeInterval, duration: TimeInterval) -> Double {
        guard duration > 0, duration.isFinite, position.isFinite else { return 0 }
        return min(1, max(0, position / duration))
    }

    static func angle(isPlaying: Bool, position: TimeInterval, duration: TimeInterval) -> Double {
        guard isPlaying else { return rest }
        return start + (end - start) * progress(position: position, duration: duration)
    }
}
