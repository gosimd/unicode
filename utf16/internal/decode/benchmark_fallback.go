//go:build !goexperiment.simd || (!amd64 && !arm64)

package decode

func AvailableForBenchmark() bool {
	return false
}

func DecodeCoreForBenchmark([]uint16, []rune) []rune {
	panic("utf16: SIMD decoder is unavailable")
}
