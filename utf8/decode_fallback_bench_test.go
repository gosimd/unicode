//go:build !goexperiment.simd || !arm64

package utf8

// DecodeSIMDBenchmarkPlan keeps the benchmark source buildable when the ARM64
// SIMD implementation is unavailable.
type DecodeSIMDBenchmarkPlan struct{}

func NewDecodeSIMDBenchmarkPlan(string) (DecodeSIMDBenchmarkPlan, int, bool) {
	return DecodeSIMDBenchmarkPlan{}, 0, false
}

func DecodeSIMDCoreForBenchmark(string, []rune, DecodeSIMDBenchmarkPlan) []rune {
	panic("utf8: ARM64 SIMD decoder is unavailable")
}
