import SwiftUI

extension Color {
    /// "#rrggbb" → couleur ; valeur invalide → accent par défaut de Vinyl.
    init(hex: String) {
        var s = hex.hasPrefix("#") ? String(hex.dropFirst()) : hex
        if s.count != 6 || UInt32(s, radix: 16) == nil { s = "f5c542" }
        let v = UInt32(s, radix: 16)!
        self.init(red: Double((v >> 16) & 0xff) / 255, green: Double((v >> 8) & 0xff) / 255,
                  blue: Double(v & 0xff) / 255)
    }
}
