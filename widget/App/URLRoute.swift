import AppKit

/// URL ouvertes par un clic sur le widget.
enum URLRoute {
    static func handle(_ url: URL) {
        guard url.scheme == "vinylwidget" else { return }
        switch url.host {
        case "music":
            if let app = NSWorkspace.shared.urlForApplication(withBundleIdentifier: MusicBridge.bundleID) {
                NSWorkspace.shared.openApplication(at: app, configuration: NSWorkspace.OpenConfiguration())
            }
        case "automation":
            if let u = URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_Automation") {
                NSWorkspace.shared.open(u)
            }
        default:
            break
        }
    }
}
