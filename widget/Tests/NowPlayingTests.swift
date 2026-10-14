import Foundation
import Testing

struct NowPlayingTests {
    let t0 = Date(timeIntervalSince1970: 1_000_000)

    func playing(position: TimeInterval = 10, duration: TimeInterval = 200) -> NowPlaying {
        NowPlaying(availability: .running, state: .playing, trackID: "A1", title: "Titre", artist: "Artiste",
                   album: "Album", duration: duration, position: position, capturedAt: t0,
                   artworkFile: "artwork-A1.jpg", accentHex: "#f5c542")
    }

    @Test func positionAvanceEnLecture() {
        #expect(playing().position(at: t0.addingTimeInterval(5)) == 15)
    }

    @Test func positionBorneeALaDuree() {
        #expect(playing(position: 198).position(at: t0.addingTimeInterval(10)) == 200)
    }

    @Test func positionFigeeEnPause() {
        var np = playing()
        np.state = .paused
        #expect(np.position(at: t0.addingTimeInterval(30)) == 10)
    }

    @Test func positionSansDureeNonBornee() {
        #expect(playing(position: 10, duration: 0).position(at: t0.addingTimeInterval(5)) == 15)
    }

    @Test func fermeGardeLeDernierMorceau() {
        let c = playing().closed(accentHex: "#112233", now: t0.addingTimeInterval(4))
        #expect(c.availability == .closed)
        #expect(c.state == .paused)
        #expect(c.title == "Titre")
        #expect(c.artworkFile == "artwork-A1.jpg")
        #expect(c.position == 14)
        #expect(c.capturedAt == t0.addingTimeInterval(4))
        #expect(c.accentHex == "#112233")
    }

    @Test func fermeSansMorceauDonneJamaisJoue() {
        let c = NowPlaying.neverPlayed(now: t0).closed(accentHex: "#112233", now: t0)
        #expect(c.availability == .neverPlayed)
        #expect(c.accentHex == "#112233")
    }
}
