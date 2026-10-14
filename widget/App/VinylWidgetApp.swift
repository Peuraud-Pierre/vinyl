import AppKit
import WidgetKit

final class AppDelegate: NSObject, NSApplicationDelegate {
    private let bridge = MusicBridge()
    private var monitor: NowPlayingMonitor?

    func applicationDidFinishLaunching(_ notification: Notification) {
        LoginItem.registerIfNeeded()
        guard let dir = AppGroup.containerURL else {
            NSLog("Vinyl Widget : App Group indisponible (%@)", AppGroup.identifier)
            return
        }
        let monitor = NowPlayingMonitor(bridge: bridge, store: StateStore(directory: dir),
                                        reload: { WidgetCenter.shared.reloadAllTimelines() })
        self.monitor = monitor
        CommandListener.start { [bridge] cmd in
            DispatchQueue.global(qos: .userInitiated).async { bridge.send(cmd) }
        }
        monitor.start()
    }

    func application(_ application: NSApplication, open urls: [URL]) {
        urls.forEach(URLRoute.handle)
    }
}

@main
enum Main {
    static let delegate = AppDelegate()

    static func main() {
        let app = NSApplication.shared
        app.delegate = delegate
        app.setActivationPolicy(.accessory)
        app.run()
    }
}
