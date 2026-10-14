import Foundation
import Testing

struct StateStoreTests {
    let dir = FileManager.default.temporaryDirectory
        .appendingPathComponent("vinylwidget-tests-\(UUID().uuidString)")

    @Test func fichierAbsent() {
        #expect(StateStore(directory: dir).load() == nil)
    }

    @Test func allerRetour() throws {
        let store = StateStore(directory: dir)
        let np = NowPlaying(availability: .running, state: .paused, trackID: "X", title: "Été 🎵", artist: "A",
                            album: "B", duration: 213.5, position: 12.25,
                            capturedAt: Date(timeIntervalSince1970: 1_234_567.891),
                            artworkFile: nil, accentHex: "#ff6b6b")
        try store.save(np)
        #expect(store.load() == np)
    }

    @Test func fichierIllisible() throws {
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        try Data("pas du json".utf8).write(to: dir.appendingPathComponent("state.json"))
        #expect(StateStore(directory: dir).load() == nil)
    }

    @Test func pochetteRemplaceLAncienne() throws {
        let store = StateStore(directory: dir)
        let a = try store.saveArtwork(Data([1]), trackID: "AAA")
        let b = try store.saveArtwork(Data([2]), trackID: "BBB")
        #expect(a == "artwork-AAA.jpg")
        #expect(b == "artwork-BBB.jpg")
        #expect(!FileManager.default.fileExists(atPath: store.artworkURL(a).path))
        #expect(try Data(contentsOf: store.artworkURL(b)) == Data([2]))
    }

    @Test func identifiantNettoye() throws {
        #expect(try StateStore(directory: dir).saveArtwork(Data([1]), trackID: "../x/../y") == "artwork-xy.jpg")
        #expect(try StateStore(directory: dir).saveArtwork(Data([1]), trackID: "") == "artwork-x.jpg")
    }
}
