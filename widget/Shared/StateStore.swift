import Foundation

/// Lecture/écriture de l'état et de la pochette dans le conteneur partagé.
struct StateStore {
    let directory: URL

    var stateURL: URL { directory.appendingPathComponent("state.json") }

    func load() -> NowPlaying? {
        guard let data = try? Data(contentsOf: stateURL) else { return nil }
        return try? JSONDecoder().decode(NowPlaying.self, from: data)
    }

    func save(_ np: NowPlaying) throws {
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        try JSONEncoder().encode(np).write(to: stateURL, options: .atomic)
    }

    func artworkURL(_ name: String) -> URL { directory.appendingPathComponent(name) }

    /// Écrit la pochette de façon atomique, supprime les autres et renvoie le nom du fichier.
    func saveArtwork(_ jpeg: Data, trackID: String) throws -> String {
        let safe = String(trackID.filter { $0.isLetter || $0.isNumber })
        let name = "artwork-\(safe.isEmpty ? "x" : safe).jpg"
        try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
        try jpeg.write(to: artworkURL(name), options: .atomic)
        let files = (try? FileManager.default.contentsOfDirectory(atPath: directory.path)) ?? []
        for f in files where f.hasPrefix("artwork-") && f != name {
            try? FileManager.default.removeItem(at: artworkURL(f))
        }
        return name
    }
}
