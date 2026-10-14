import AppKit

/// Écoute Musique et publie l'état pour le widget. Toutes les méthodes s'appellent sur le fil principal.
final class NowPlayingMonitor {
    private let bridge: MusicBridge
    private let store: StateStore
    private let reload: () -> Void
    private let queue = DispatchQueue(label: "com.peuraud.vinylwidget.music")
    private var current: NowPlaying
    private var busy = false
    private var pending = false
    private var observers: [NSObjectProtocol] = []
    private var timer: Timer?

    init(bridge: MusicBridge, store: StateStore, reload: @escaping () -> Void) {
        self.bridge = bridge
        self.store = store
        self.reload = reload
        current = store.load() ?? .neverPlayed(accentHex: AccentReader.read())
    }

    func start() {
        observers.append(DistributedNotificationCenter.default().addObserver(
            forName: Notification.Name("com.apple.Music.playerInfo"), object: nil, queue: .main
        ) { [weak self] _ in self?.refresh() })

        let ws = NSWorkspace.shared.notificationCenter
        observers.append(ws.addObserver(forName: NSWorkspace.didLaunchApplicationNotification,
                                        object: nil, queue: .main) { [weak self] note in
            if Self.isMusic(note) { self?.refresh() }
        })
        // À la fermeture, aucun AppleScript : Musique serait relancée.
        observers.append(ws.addObserver(forName: NSWorkspace.didTerminateApplicationNotification,
                                        object: nil, queue: .main) { [weak self] note in
            guard let self, Self.isMusic(note) else { return }
            self.publish(self.current.closed(accentHex: AccentReader.read(), now: Date()))
        })

        timer = Timer.scheduledTimer(withTimeInterval: 10, repeats: true) { [weak self] _ in
            guard let self, self.current.availability == .running, self.current.state == .playing else { return }
            self.refresh()
        }
        refresh()
    }

    func refresh() {
        if busy { pending = true; return }
        busy = true
        let previous = current
        queue.async { [bridge, store] in
            let next = StateReader.read(previous: previous, bridge: bridge, store: store, now: Date())
            DispatchQueue.main.async {
                self.busy = false
                if let next { self.publish(next) }
                if self.pending {
                    self.pending = false
                    self.refresh()
                }
            }
        }
    }

    private func publish(_ next: NowPlaying) {
        guard StateDiff.shouldPublish(old: current, new: next) else { return }
        do {
            try store.save(next)
        } catch {
            NSLog("Vinyl Widget : écriture de l'état impossible : %@", "\(error)")
            return
        }
        current = next
        reload()
    }

    private static func isMusic(_ note: Notification) -> Bool {
        (note.userInfo?[NSWorkspace.applicationUserInfoKey] as? NSRunningApplication)?.bundleIdentifier
            == MusicBridge.bundleID
    }
}
