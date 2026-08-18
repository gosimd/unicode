//go:build goexperiment.simd && amd64

package decode

import "simd/archsimd"

func AvailableForBenchmark() bool {
	return archsimd.X86.AVX2()
}

// DecodeCoreForBenchmark measures SIMD decoding without result allocation.
// out must have room for len(s) runes.
func DecodeCoreForBenchmark(s []uint16, out []rune) []rune {
	if !AvailableForBenchmark() {
		panic("utf16: amd64 SIMD decoder is unavailable")
	}
	return decodeSIMD(s, out)
}

func DecodeAVX2CoreForBenchmark(s []uint16, out []rune) []rune {
	return decodeAVX2(s, out)
}

func DecodeAVX512CoreForBenchmark(s []uint16, out []rune) []rune {
	return decodeAVX512(s, out)
}
