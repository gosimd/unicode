//go:build !goexperiment.simd || (!amd64 && !arm64)

package encode

func AvailableForBenchmark() bool {
	return false
}

func EncodeCoreForBenchmark([]rune, []uint16) []uint16 {
	panic("utf16: SIMD encoder is unavailable")
}
