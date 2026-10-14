import SwiftUI
import WidgetKit

struct WidgetRootView: View {
    @Environment(\.widgetFamily) private var family
    let entry: VinylEntry

    var body: some View {
        content
            .containerBackground(for: .widget) { Color(red: 0.08, green: 0.08, blue: 0.08) }
            .widgetURL(entry.nowPlaying.availability == .permissionDenied ? WidgetLinks.automation : WidgetLinks.music)
    }

    @ViewBuilder private var content: some View {
        switch family {
        case .systemSmall: SmallView(entry: entry)
        case .systemMedium: MediumView(entry: entry)
        default: LargeView(entry: entry)
        }
    }
}

struct SmallView: View {
    let entry: VinylEntry

    var body: some View {
        ZStack(alignment: .bottom) {
            TurntableView(entry: entry, playButton: true).padding(8)
            if let s = statusText(entry.nowPlaying) {
                Text(s).font(.caption2).foregroundStyle(.white.opacity(0.8)).lineLimit(2)
                    .multilineTextAlignment(.center).padding(.horizontal, 8).padding(.bottom, 6)
            }
        }
    }
}

struct MediumView: View {
    let entry: VinylEntry

    var body: some View {
        HStack(spacing: 12) {
            TurntableView(entry: entry).frame(maxWidth: .infinity)
            VStack(alignment: .leading, spacing: 10) {
                InfoView(entry: entry)
                ProgressBarView(entry: entry)
                ControlsView(entry: entry)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(14)
    }
}

struct LargeView: View {
    let entry: VinylEntry

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            TurntableView(entry: entry)
            InfoView(entry: entry)
            ProgressBarView(entry: entry)
            ControlsView(entry: entry).frame(maxWidth: .infinity)
        }
        .padding(16)
    }
}
