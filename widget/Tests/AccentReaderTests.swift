import Foundation
import Testing

struct AccentReaderTests {
    let dir = FileManager.default.temporaryDirectory.appendingPathComponent("vinylwidget-accent-\(UUID().uuidString)")

    func write(_ json: String) throws -> URL {
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        let url = dir.appendingPathComponent("settings.json")
        try Data(json.utf8).write(to: url)
        return url
    }

    @Test func fichierAbsent() {
        #expect(AccentReader.read(from: dir.appendingPathComponent("absent.json")) == "#f5c542")
    }

    @Test func couleurValide() throws {
        #expect(AccentReader.read(from: try write(##"{"accent":"#FF6b6b","texture":"black"}"##)) == "#FF6b6b")
    }

    @Test func couleurInvalide() throws {
        #expect(AccentReader.read(from: try write(##"{"accent":"#zzz"}"##)) == "#f5c542")
        #expect(AccentReader.read(from: try write(#"{"accent":42}"#)) == "#f5c542")
        #expect(AccentReader.read(from: try write("[1,2]")) == "#f5c542")
    }
}
