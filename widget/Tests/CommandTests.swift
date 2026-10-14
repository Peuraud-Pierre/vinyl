import Testing

struct CommandTests {
    @Test func nomsDesNotifications() {
        #expect(Command.playpause.notificationName == "com.peuraud.vinylwidget.playpause")
        #expect(Command.next.notificationName == "com.peuraud.vinylwidget.next")
        #expect(Command.previous.notificationName == "com.peuraud.vinylwidget.previous")
    }

    @Test func allerRetour() {
        for c in Command.allCases {
            #expect(Command(notificationName: c.notificationName) == c)
        }
        #expect(Command(notificationName: "com.apple.Music.playerInfo") == nil)
        #expect(Command(notificationName: "com.peuraud.vinylwidget.stop") == nil)
    }
}
