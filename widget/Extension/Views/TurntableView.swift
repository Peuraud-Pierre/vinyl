import SwiftUI

/// Disque + bras, à l'échelle de la place disponible (rapport 420×320 conservé).
struct TurntableView: View {
    let entry: VinylEntry
    var playButton = false

    var body: some View {
        GeometryReader { geo in
            let s = min(geo.size.width / Geometry.width, geo.size.height / Geometry.height)
            ZStack(alignment: .topLeading) {
                DiscView(entry: entry, diameter: Geometry.discRadius * 2 * s, playButton: playButton)
                    .offset(x: (Geometry.discCenter.x - Geometry.discRadius) * s,
                            y: (Geometry.discCenter.y - Geometry.discRadius) * s)
                TonearmView(angle: entry.armAngle, accent: Color(hex: entry.nowPlaying.accentHex), scale: s)
                    .allowsHitTesting(false)
            }
            .frame(width: Geometry.width * s, height: Geometry.height * s, alignment: .topLeading)
            .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .opacity(entry.nowPlaying.availability == .closed ? 0.55 : 1)
    }
}
