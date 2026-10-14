import Foundation
import ServiceManagement

enum LoginItem {
    /// Inscrit l'app pour qu'elle démarre à chaque ouverture de session.
    static func registerIfNeeded() {
        let service = SMAppService.mainApp
        guard service.status != .enabled else { return }
        do { try service.register() } catch {
            NSLog("Vinyl Widget : démarrage à l'ouverture de session impossible : %@", "\(error)")
        }
    }
}
