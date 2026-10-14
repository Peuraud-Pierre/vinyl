import AppIntents
import SwiftUI
import WidgetKit

/// Étiquette centrale : pochette, ou couleur d'accent avec une note de musique.
struct LabelView: View {
    let entry: VinylEntry

    private var artwork: NSImage? {
        guard let name = entry.nowPlaying.artworkFile, let dir = AppGroup.containerURL else { return nil }
        return NSImage(contentsOf: StateStore(directory: dir).artworkURL(name))
    }

    var body: some View {
        if let img = artwork {
            Image(nsImage: img).resizable().scaledToFill().clipShape(Circle())
        } else {
            Circle().fill(Color(hex: entry.nowPlaying.accentHex))
                .overlay(Image(systemName: "music.note").font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(.black.opacity(0.6)))
        }
    }
}

/// Disque noir : sillons, reflet, étiquette, trou central. `playButton` fait de l'étiquette un bouton ⏯.
struct DiscView: View {
    let entry: VinylEntry
    let diameter: CGFloat
    var playButton = false

    var body: some View {
        ZStack {
            Circle().fill(Color(white: 0.06))
            ForEach(0..<14, id: \.self) { i in
                Circle().stroke(Color.white.opacity(0.05), lineWidth: 0.6)
                    .padding(diameter * (0.04 + CGFloat(i) * 0.022))
            }
            Circle().fill(AngularGradient(colors: [.clear, .white.opacity(0.10), .clear, .clear,
                                                   .white.opacity(0.07), .clear], center: .center))
            label.frame(width: diameter * Geometry.labelRatio, height: diameter * Geometry.labelRatio)
            Circle().fill(Color.black).frame(width: diameter * 0.025, height: diameter * 0.025)
        }
        .frame(width: diameter, height: diameter)
    }

    @ViewBuilder private var label: some View {
        if playButton {
            Button(intent: PlayPauseIntent()) { LabelView(entry: entry) }.buttonStyle(.plain)
        } else {
            LabelView(entry: entry)
        }
    }
}
