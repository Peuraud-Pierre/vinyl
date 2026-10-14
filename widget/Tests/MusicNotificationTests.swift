import Foundation
import Testing

struct MusicNotificationTests {
    let t0 = Date(timeIntervalSince1970: 1_000_000)

    @Test func etatArreteDetecte() {
        #expect(MusicNotification.isStopped(["Player State": "Stopped"]))
        #expect(!MusicNotification.isStopped(["Player State": "Playing"]))
        #expect(!MusicNotification.isStopped(["Player State": "Paused"]))
        #expect(!MusicNotification.isStopped([:]))
        #expect(!MusicNotification.isStopped(nil))
    }

    @Test func arretSansAppleScriptGardeLeMorceau() {
        let np = NowPlaying(availability: .running, state: .playing, trackID: "A", title: "T", artist: "A", album: "B",
                            duration: 200, position: 10, capturedAt: t0, artworkFile: "artwork-A.jpg", accentHex: "#f5c542")
        let s = np.stopped(accentHex: "#112233", now: t0.addingTimeInterval(5))
        #expect(s.availability == .running)
        #expect(s.state == .stopped)
        #expect(s.position == 0)
        #expect(s.title == "T")
        #expect(s.accentHex == "#112233")
        #expect(s.capturedAt == t0.addingTimeInterval(5))
    }

    @Test func arretSansMorceauDonneJamaisJoue() {
        #expect(NowPlaying.neverPlayed(now: t0).stopped(accentHex: "#112233", now: t0).availability == .neverPlayed)
    }
}
