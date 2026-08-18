//go:build !goexperiment.simd || (!arm64 && !amd64)

package decode

// DecodeSIMDBenchmarkPlan keeps the benchmark source buildable when a SIMD
// decoder is unavailable.
type DecodeSIMDBenchmarkPlan struct{}

func NewDecodeSIMDBenchmarkPlan(string) (DecodeSIMDBenchmarkPlan, int, bool) {
	return DecodeSIMDBenchmarkPlan{}, 0, false
}

func DecodeSIMDCoreForBenchmark(string, []rune, DecodeSIMDBenchmarkPlan) []rune {
	panic("utf8: SIMD decoder is unavailable")
}
