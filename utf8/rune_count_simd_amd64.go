//go:build goexperiment.simd && amd64

package utf8

import "simd/archsimd"

func runeCountSIMD(p []byte) (int, bool) {
	if archsimd.X86.AVX512() {
		return runeCountAVX512(p)
	}
	return runeCountAVX2(p)
}
