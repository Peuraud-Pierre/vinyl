import Foundation

/// Ordres envoyés par le widget à l'assistant, par notification Darwin (autorisée dans le bac à sable).
enum Command: String, CaseIterable {
    case playpause, next, previous

    static let prefix = "com.peuraud.vinylwidget."

    var notificationName: String { Self.prefix + rawValue }

    init?(notificationName: String) {
        guard notificationName.hasPrefix(Self.prefix) else { return nil }
        self.init(rawValue: String(notificationName.dropFirst(Self.prefix.count)))
    }

    func post() {
        CFNotificationCenterPostNotification(CFNotificationCenterGetDarwinNotifyCenter(),
                                             CFNotificationName(notificationName as CFString), nil, nil, true)
    }
}
