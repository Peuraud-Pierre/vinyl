import SwiftUI

/// Bras de lecture : angle 0 = vertical vers le bas depuis le pivot, positif vers le disque.
struct TonearmView: View {
    let angle: Double
    let accent: Color
    let scale: CGFloat

    var body: some View {
        Canvas { ctx, _ in
            let s = scale
            let pivot = CGPoint(x: Geometry.pivot.x * s, y: Geometry.pivot.y * s)
            let rad = angle * .pi / 180
            let tip = CGPoint(x: pivot.x - sin(rad) * Geometry.armLength * s,
                              y: pivot.y + cos(rad) * Geometry.armLength * s)
            let back = CGPoint(x: pivot.x + sin(rad) * 20 * s, y: pivot.y - cos(rad) * 20 * s)
            var arm = Path()
            arm.move(to: back)
            arm.addLine(to: tip)
            ctx.stroke(arm, with: .color(Color(white: 0.78)), style: StrokeStyle(lineWidth: 5 * s, lineCap: .round))
            ctx.fill(Path(ellipseIn: CGRect(x: back.x - 11 * s, y: back.y - 11 * s, width: 22 * s, height: 22 * s)),
                     with: .color(accent))
            ctx.fill(Path(ellipseIn: CGRect(x: pivot.x - 16 * s, y: pivot.y - 16 * s, width: 32 * s, height: 32 * s)),
                     with: .color(Color(white: 0.3)))
            ctx.fill(Path(roundedRect: CGRect(x: tip.x - 7 * s, y: tip.y - 10 * s, width: 14 * s, height: 20 * s),
                          cornerRadius: 3 * s), with: .color(Color(white: 0.85)))
        }
    }
}
