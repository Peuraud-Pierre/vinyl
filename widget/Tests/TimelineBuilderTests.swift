import Foundation
import Testing

struct TimelineBuilderTests {
    let t0 = Date(timeIntervalSince1970: 1_000_000)

    func np(_ state: NowPlaying.PlayerState = .playing, availability: NowPlaying.Availability = .running,
            position: TimeInterval = 0, duration: TimeInterval = 60) -> NowPlaying {
        NowPlaying(availability: availability, state: state, trackID: "A", title: "T", artist: "A", album: "B",
                   duration: duration, position: position, capturedAt: t0, artworkFile: nil, accentHex: "#f5c542")
    }

    @Test func lectureUneEntreeToutesLes15s() {
        let r = TimelineBuilder.entries(for: np(), now: t0)
        #expect(r.entries.map(\.position) == [0, 15, 30, 45, 60])
        #expect(r.entries.map(\.date) == [0, 15, 30, 45, 60].map { t0.addingTimeInterval($0) })
        #expect(r.refreshAtEnd)
    }

    @Test func positionCalculeeDepuisLaCapture() {
        let r = TimelineBuilder.entries(for: np(position: 10), now: t0.addingTimeInterval(5))
        #expect(r.entries.first?.position == 15)
    }

    @Test func pauseUneSeuleEntree() {
        let r = TimelineBuilder.entries(for: np(.paused, position: 20), now: t0)
        #expect(r.entries.count == 1)
        #expect(r.entries[0].armAngle == ArmMath.rest)
        #expect(!r.refreshAtEnd)
    }

    @Test func musiqueFermeeUneSeuleEntree() {
        let r = TimelineBuilder.entries(for: np(availability: .closed), now: t0)
        #expect(r.entries.count == 1)
        #expect(!r.refreshAtEnd)
    }

    @Test func dureeNulle() {
        let r = TimelineBuilder.entries(for: np(duration: 0), now: t0)
        #expect(r.entries.count == 1)
        #expect(r.entries[0].armAngle == ArmMath.start)
        #expect(!r.refreshAtEnd)
    }

    @Test func lectureAuDelaDeLaFin() {
        let r = TimelineBuilder.entries(for: np(position: 50), now: t0.addingTimeInterval(120))
        #expect(r.entries.count == 1)
        #expect(!r.refreshAtEnd)
    }

    @Test func morceauLongPlafonne() {
        let r = TimelineBuilder.entries(for: np(duration: 3600), now: t0)
        #expect(r.entries.count == TimelineBuilder.maxEntries)
        #expect(r.refreshAtEnd)
    }
}
