package pow

// Known values for salt "00112233445566778899aabb?expires=1790000000&issued=1789998200",
// number 4242 and key "test-key"; ported from old_simon-ostendorf.de, whose
// jstest/worker.test.cjs checks the same values against the browser worker.
const (
	knownChallenge = "1f5e37a25a70f955cff381b03a1d9937bbe9471796bb25f057d1d795403035ab"
	knownSignature = "98da1c93f6c4f169927d0c939860a4f3dec48524b4220ad51d337ad63ec1aa5a"
)
