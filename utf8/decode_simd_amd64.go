//go:build goexperiment.simd && amd64

package utf8

import (
	"simd/archsimd"
	"unsafe"
)

// Decode returns the runes in s. Invalid UTF-8 encodings are replaced by
// RuneError, as in the language conversion []rune(s).
func Decode(s string) []rune {
	if len(s) == 0 || !archsimd.X86.AVX2() {
		return []rune(s)
	}

	input := unsafe.Slice(unsafe.StringData(s), len(s))
	count, valid := runeCountSIMD(input)
	if !valid {
		return []rune(s)
	}

	out := make([]rune, count)
	return decodeValidSIMD(input, out, count)
}

// decodeValidSIMD dispatches an already-validated input to the best available
// amd64 decoder. The caller owns out, which must contain outputRunes lanes.
func decodeValidSIMD(input []byte, out []rune, outputRunes int) []rune {
	if archsimd.X86.AVX512() {
		return decodeValidAVX512(input, out, outputRunes)
	}
	return decodeValidAVX2(input, out, outputRunes)
}
