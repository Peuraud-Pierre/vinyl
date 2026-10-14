import AppIntents

// L'extension ne parle jamais à Musique : elle prévient l'assistant.
struct PlayPauseIntent: AppIntent {
    static var title: LocalizedStringResource = "Lecture / pause"
    func perform() async throws -> some IntentResult {
        Command.playpause.post()
        return .result()
    }
}

struct NextIntent: AppIntent {
    static var title: LocalizedStringResource = "Morceau suivant"
    func perform() async throws -> some IntentResult {
        Command.next.post()
        return .result()
    }
}

struct PreviousIntent: AppIntent {
    static var title: LocalizedStringResource = "Morceau précédent"
    func perform() async throws -> some IntentResult {
        Command.previous.post()
        return .result()
    }
}
