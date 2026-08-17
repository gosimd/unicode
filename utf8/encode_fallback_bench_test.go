//go:build !goexperiment.simd || !arm64

package utf8

// EncodeSIMDBenchmarkPlan keeps the benchmark source buildable when the ARM64
// SIMD implementation is unavailable.
type EncodeSIMDBenchmarkPlan struct{}

func NewEncodeSIMDBenchmarkPlan([]rune) (EncodeSIMDBenchmarkPlan, int, bool) {
	return EncodeSIMDBenchmarkPlan{}, 0, false
}

func EncodeSIMDCoreForBenchmark([]rune, []byte, EncodeSIMDBenchmarkPlan) []byte {
	panic("utf8: ARM64 SIMD encoder is unavailable")
}
