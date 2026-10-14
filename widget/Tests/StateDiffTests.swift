import Foundation
import Testing

struct StateDiffTests {
    let t0 = Date(timeIntervalSince1970: 1_000_000)

    func np(_ state: NowPlaying.PlayerState = .playing, track: String = "A", position: TimeInterval,
            at offset: TimeInterval = 0, accent: String = "#f5c542") -> NowPlaying {
        NowPlaying(availability: .running, state: state, trackID: track, title: "T", artist: "A", album: "B",
                   duration: 200, position: position, capturedAt: t0.addingTimeInterval(offset),
                   artworkFile: nil, accentHex: accent)
    }

    @Test func lectureNormaleNePublieRien() {
        #expect(!StateDiff.shouldPublish(old: np(position: 10), new: np(position: 15, at: 5)))
        #expect(!StateDiff.shouldPublish(old: np(position: 10), new: np(position: 17.5, at: 5)))
    }

    @Test func deplacementDansLeMorceau() {
        #expect(StateDiff.shouldPublish(old: np(position: 10), new: np(position: 60, at: 5)))
    }

    @Test func changementDeMorceau() {
        #expect(StateDiff.shouldPublish(old: np(position: 10), new: np(track: "B", position: 0, at: 5)))
    }

    @Test func pauseVersLecture() {
        #expect(StateDiff.shouldPublish(old: np(.paused, position: 10), new: np(position: 10, at: 5)))
    }

    @Test func couleurDAccent() {
        #expect(StateDiff.shouldPublish(old: np(position: 10), new: np(position: 10, accent: "#112233")))
    }

    @Test func deplacementEnPause() {
        #expect(!StateDiff.shouldPublish(old: np(.paused, position: 10), new: np(.paused, position: 10, at: 30)))
        #expect(StateDiff.shouldPublish(old: np(.paused, position: 10), new: np(.paused, position: 50, at: 30)))
    }
}
