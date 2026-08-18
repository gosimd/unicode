//go:build goexperiment.simd && amd64

package encode

import "simd/archsimd"

func AvailableForBenchmark() bool {
	return archsimd.X86.AVX2()
}

// EncodeCoreForBenchmark measures planning and SIMD encoding without output
// allocation. out must have room for twice as many code units as s has runes.
func EncodeCoreForBenchmark(s []rune, out []uint16) []uint16 {
	if !AvailableForBenchmark() {
		panic("utf16: amd64 SIMD encoder is unavailable")
	}
	plan := planEncodeSIMD(s)
	return encodeSIMDWithPlan(s, out, plan)
}

func EncodeAVX2CoreForBenchmark(s []rune, out []uint16) []uint16 {
	capacity, mode := encodedLengthAVX2Profile(s)
	return encodeAVX2(s, out, capacity, mode)
}

func EncodeAVX512CoreForBenchmark(s []rune, out []uint16) []uint16 {
	capacity, mode := encodedLengthAVX512Profile(s)
	return encodeAVX512(s, out, capacity, mode)
}
