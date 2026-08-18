//go:build goexperiment.simd && arm64

package decode

func AvailableForBenchmark() bool {
	return true
}

// DecodeCoreForBenchmark measures SIMD decoding without result allocation.
// out must have room for len(s) runes.
func DecodeCoreForBenchmark(s []uint16, out []rune) []rune {
	return decodeSIMD(s, out)
}
