//go:build goexperiment.simd && arm64

package scan

// CountValid validates p and counts its runes in one SIMD pass.
func CountValid(p []byte) (int, bool) {
	return runeCountSIMD128(p)
}
