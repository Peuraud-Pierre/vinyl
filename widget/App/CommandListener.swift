import Foundation

/// Reçoit les ordres du widget (notifications Darwin) et les transmet au fil principal.
enum CommandListener {
    private static var handler: ((Command) -> Void)?

    static func start(_ h: @escaping (Command) -> Void) {
        handler = h
        let center = CFNotificationCenterGetDarwinNotifyCenter()
        for c in Command.allCases {
            CFNotificationCenterAddObserver(center, nil, { _, _, name, _, _ in
                guard let cf = name?.rawValue, let cmd = Command(notificationName: cf as String) else { return }
                DispatchQueue.main.async { CommandListener.handler?(cmd) }
            }, c.notificationName as CFString, nil, .deliverImmediately)
        }
    }
}
