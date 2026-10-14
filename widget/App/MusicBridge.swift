import AppKit

enum BridgeError: Error {
    case permissionDenied
    case timeout
    case failed(String)
}

/// Dialogue avec Musique par /usr/bin/osascript. Ne lance jamais Musique, sauf sur ⏯ (send(.playpause)).
final class MusicBridge {
    static let bundleID = "com.apple.Music"

    /// Une app en cours de fermeture ne compte pas : lui parler la relancerait.
    var isRunning: Bool {
        NSRunningApplication.runningApplications(withBundleIdentifier: Self.bundleID).contains { !$0.isTerminated }
    }

    private static let stateScript = """
    if application id "com.apple.Music" is not running then return "closed"
    set US to (ASCII character 31)
    tell application id "com.apple.Music"
        set ps to player state as text
        if ps is "stopped" then return "stopped"
        set t to current track
        return ps & US & (persistent ID of t) & US & (name of t) & US & (artist of t) & US & (album of t) & US & ((duration of t) as text) & US & (player position as text)
    end tell
    """

    private static let artworkScript = """
    on run argv
        if application id "com.apple.Music" is not running then return "none"
        tell application id "com.apple.Music"
            if (persistent ID of current track) is not (item 2 of argv) then return "changed"
            if (count of artworks of current track) is 0 then return "none"
            set d to raw data of artwork 1 of current track
        end tell
        set f to open for access (POSIX file (item 1 of argv)) with write permission
        try
            set eof f to 0
            write d to f
        end try
        close access f
        return "ok"
    end run
    """

    func readState() -> Result<MusicReading, BridgeError> {
        guard isRunning else { return .success(.closed) }
        return run(Self.stateScript).flatMap { out -> Result<MusicReading, BridgeError> in
            guard let reading = MusicParser.parse(out) else { return .failure(.failed("réponse inattendue : \(out)")) }
            return .success(reading)
        }
    }

    /// Écrit les octets bruts de la pochette du morceau `trackID` dans `url`.
    /// false si pas de pochette ou si le morceau a changé entre-temps.
    func writeArtwork(to url: URL, trackID: String) -> Bool {
        guard isRunning, case .success(let out) = run(Self.artworkScript, args: [url.path, trackID]) else { return false }
        return out.trimmingCharacters(in: .whitespacesAndNewlines) == "ok"
    }

    func send(_ command: Command) {
        switch command {
        case .playpause:
            if isRunning {
                _ = run(#"tell application id "com.apple.Music" to playpause"#)
            } else {
                _ = run(#"tell application id "com.apple.Music" to play"#, timeout: 15) // lance Musique
            }
        case .next:
            guard isRunning else { return }
            _ = run(#"tell application id "com.apple.Music" to next track"#)
        case .previous:
            guard isRunning else { return }
            _ = run(#"tell application id "com.apple.Music" to previous track"#)
        }
    }

    private func run(_ script: String, args: [String] = [], timeout: TimeInterval = 2) -> Result<String, BridgeError> {
        let p = Process()
        p.executableURL = URL(fileURLWithPath: "/usr/bin/osascript")
        p.arguments = ["-e", script, "--"] + args
        let out = Pipe(), err = Pipe()
        p.standardOutput = out
        p.standardError = err
        let done = DispatchSemaphore(value: 0)
        p.terminationHandler = { _ in done.signal() }
        do { try p.run() } catch { return .failure(.failed(error.localizedDescription)) }
        if done.wait(timeout: .now() + timeout) == .timedOut {
            p.terminate()
            return .failure(.timeout)
        }
        let stdout = String(data: out.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""
        let stderr = String(data: err.fileHandleForReading.readDataToEndOfFile(), encoding: .utf8) ?? ""
        guard p.terminationStatus == 0 else {
            return .failure(stderr.contains("-1743") ? .permissionDenied : .failed(stderr))
        }
        return .success(stdout)
    }
}
