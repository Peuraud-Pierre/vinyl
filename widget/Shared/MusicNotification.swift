import Foundation

/// Lecture de la notification distribuée `com.apple.Music.playerInfo`.
enum MusicNotification {
    /// Musique envoie « Stopped » à l'arrêt, et aussi pendant qu'elle se ferme :
    /// dans ce cas, aucun AppleScript, sinon Musique serait relancée.
    static func isStopped(_ userInfo: [AnyHashable: Any]?) -> Bool {
        (userInfo?["Player State"] as? String) == "Stopped"
    }
}
