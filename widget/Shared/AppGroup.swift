import Foundation

/// Conteneur partagé entre l'assistant et l'extension.
enum AppGroup {
    /// Valeur de la clé Info.plist `VinylAppGroup` (`$(TeamIdentifierPrefix)com.peuraud.vinylwidget`).
    static var identifier: String {
        Bundle.main.object(forInfoDictionaryKey: "VinylAppGroup") as? String ?? ""
    }

    static var containerURL: URL? {
        guard !identifier.isEmpty else { return nil }
        return FileManager.default.containerURL(forSecurityApplicationGroupIdentifier: identifier)
    }
}
