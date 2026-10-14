import SwiftUI
import WidgetKit

extension VinylEntry: TimelineEntry {}

@main
struct VinylWidgetBundle: WidgetBundle {
    var body: some Widget { VinylWidget() }
}

struct VinylWidget: Widget {
    var body: some WidgetConfiguration {
        StaticConfiguration(kind: "VinylWidget", provider: Provider()) { entry in
            WidgetRootView(entry: entry)
        }
        .configurationDisplayName("Vinyl")
        .description("La platine Vinyl pour Apple Music.")
        .supportedFamilies([.systemSmall, .systemMedium, .systemLarge])
        .contentMarginsDisabled()
    }
}
