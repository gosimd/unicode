//go:build goexperiment.simd && amd64

package scan

import "simd/archsimd"

// CountValid validates p and counts its runes in one SIMD pass. The caller
// must check AVX2 availability before calling it on amd64.
func CountValid(p []byte) (int, bool) {
	if archsimd.X86.AVX512() {
		return runeCountAVX512(p)
	}
	return runeCountAVX2(p)
}
