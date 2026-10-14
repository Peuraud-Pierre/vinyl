import CoreGraphics

/// Géométrie du socle de Vinyl (frontend/src/turntable/arm.ts, ARM_GEOMETRY), en points sur 420×320.
enum Geometry {
    static let width: CGFloat = 420
    static let height: CGFloat = 320
    static let pivot = CGPoint(x: 324, y: 24)
    static let armLength: CGFloat = 222
    static let discCenter = CGPoint(x: 162, y: 160)
    static let discRadius: CGFloat = 132
    static let labelRatio: CGFloat = 0.36
}
