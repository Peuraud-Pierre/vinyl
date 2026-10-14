import AppIntents
import SwiftUI
import WidgetKit

enum WidgetLinks {
    static let music = URL(string: "vinylwidget://music")!
    static let automation = URL(string: "vinylwidget://automation")!
}

/// Message d'état, ou nil si Musique tourne normalement.
func statusText(_ np: NowPlaying) -> String? {
    switch np.availability {
    case .neverPlayed: return "Lance un morceau dans Musique"
    case .permissionDenied: return "Autorise Vinyl Widget à piloter Musique"
    case .closed: return "Musique fermée"
    case .running: return nil
    }
}

struct InfoView: View {
    let entry: VinylEntry

    var body: some View {
        let np = entry.nowPlaying
        VStack(alignment: .leading, spacing: 2) {
            if np.availability == .neverPlayed || np.availability == .permissionDenied {
                Text(statusText(np) ?? "").font(.headline).foregroundStyle(.white).lineLimit(2)
            } else {
                Text(np.title ?? "").font(.headline).foregroundStyle(.white).lineLimit(1)
                Text(np.artist ?? "").font(.subheadline).foregroundStyle(.white.opacity(0.6)).lineLimit(1)
                if let s = statusText(np) {
                    Text(s).font(.caption).foregroundStyle(.white.opacity(0.45))
                }
            }
        }
    }
}

struct ProgressBarView: View {
    let entry: VinylEntry

    var body: some View {
        GeometryReader { g in
            ZStack(alignment: .leading) {
                Capsule().fill(.white.opacity(0.15))
                Capsule().fill(Color(hex: entry.nowPlaying.accentHex)).frame(width: g.size.width * entry.progress)
            }
        }
        .frame(height: 4)
        .widgetAccentable()
    }
}

struct ControlsView: View {
    let entry: VinylEntry

    var body: some View {
        HStack(spacing: 22) {
            Button(intent: PreviousIntent()) { Image(systemName: "backward.fill") }
            Button(intent: PlayPauseIntent()) {
                Image(systemName: entry.isPlaying ? "pause.fill" : "play.fill").font(.title2)
            }
            Button(intent: NextIntent()) { Image(systemName: "forward.fill") }
        }
        .buttonStyle(.plain)
        .foregroundStyle(.white)
        .widgetAccentable()
    }
}
