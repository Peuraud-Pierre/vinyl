import Testing

struct ArmMathTests {
    @Test func debutMilieuFin() {
        #expect(ArmMath.angle(isPlaying: true, position: 0, duration: 200) == 18)
        #expect(ArmMath.angle(isPlaying: true, position: 100, duration: 200) == 26)
        #expect(ArmMath.angle(isPlaying: true, position: 200, duration: 200) == 34)
    }

    @Test func leveHorsLecture() {
        #expect(ArmMath.angle(isPlaying: false, position: 100, duration: 200) == 4)
    }

    @Test func bornes() {
        #expect(ArmMath.angle(isPlaying: true, position: -5, duration: 200) == 18)
        #expect(ArmMath.angle(isPlaying: true, position: 500, duration: 200) == 34)
    }

    @Test func dureeNulleOuInvalide() {
        #expect(ArmMath.angle(isPlaying: true, position: 50, duration: 0) == 18)
        #expect(ArmMath.progress(position: .nan, duration: 200) == 0)
        #expect(ArmMath.progress(position: 50, duration: .nan) == 0)
    }
}
