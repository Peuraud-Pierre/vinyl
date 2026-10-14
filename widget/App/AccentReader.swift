import Foundation

/// Couleur d'accent choisie dans Vinyl (champ `accent` de ses réglages).
enum AccentReader {
    static var vinylSettingsURL: URL {
        FileManager.default.homeDirectoryForCurrentUser
            .appendingPathComponent("Library/Application Support/Vinyl/settings.json")
    }

    static func read(from url: URL = vinylSettingsURL) -> String {
        guard let data = try? Data(contentsOf: url),
              let obj = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
              let hex = obj["accent"] as? String,
              hex.range(of: "^#[0-9a-fA-F]{6}$", options: .regularExpression) != nil
        else { return NowPlaying.defaultAccent }
        return hex
    }
}
