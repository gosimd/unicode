//go:build goexperiment.simd && amd64

package decode

import (
	"simd/archsimd"
)

// decodeSIMD dispatches to the AVX-512 or AVX2 implementation. The caller
// owns out, which must have room for len(s) runes.
func decodeSIMD(s []uint16, out []rune) []rune {
	if archsimd.X86.AVX512() {
		return decodeAVX512(s, out)
	}
	return decodeAVX2(s, out)
}
