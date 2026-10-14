import Testing

struct MusicParserTests {
    let us = MusicParser.separator

    @Test func fermeeEtArretee() {
        #expect(MusicParser.parse("closed\n") == .closed)
        #expect(MusicParser.parse("stopped") == .stopped)
    }

    @Test func morceauAvecVirgulesDecimales() {
        let out = ["playing", "ABC123", "Titre", "Artiste", "Album", "213,5", "12,25"].joined(separator: us) + "\n"
        #expect(MusicParser.parse(out) == .track(TrackInfo(state: .playing, trackID: "ABC123", title: "Titre",
                                                           artist: "Artiste", album: "Album",
                                                           duration: 213.5, position: 12.25)))
    }

    @Test func champsVidesEtMissingValue() {
        let out = ["paused", "ID", "", "", "", "missing value", "0"].joined(separator: us)
        #expect(MusicParser.parse(out) == .track(TrackInfo(state: .paused, trackID: "ID", title: "", artist: "",
                                                           album: "", duration: 0, position: 0)))
    }

    @Test func retourALaLigneEtEmoji() {
        let out = ["playing", "ID", "Ligne 1\nLigne 2 🎵", "A", "B", "100", "1"].joined(separator: us) + "\n"
        guard case .track(let t)? = MusicParser.parse(out) else { Issue.record("pas un morceau"); return }
        #expect(t.title == "Ligne 1\nLigne 2 🎵")
        #expect(t.album == "B")
        #expect(t.position == 1)
    }

    @Test func reponsesInattendues() {
        #expect(MusicParser.parse("fast forwarding" + us + "ID" + us + "T" + us + "A" + us + "B" + us + "1" + us + "1") == nil)
        #expect(MusicParser.parse(["playing", "ID", "T"].joined(separator: us)) == nil)
        #expect(MusicParser.parse("") == nil)
    }
}

struct MusicParserMissingValueTests {
    let us = MusicParser.separator

    @Test func missingValueDevientVide() {
        let out = ["playing", "ID", "missing value", "missing value", "missing value", "100", "1"].joined(separator: us)
        guard case .track(let t)? = MusicParser.parse(out) else { Issue.record("pas un morceau"); return }
        #expect(t.title == "")
        #expect(t.artist == "")
        #expect(t.album == "")
    }
}
