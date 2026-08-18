//go:build goexperiment.simd && arm64

package encode

// EncodeSIMDBenchmarkPlan is an opaque, test-only plan for benchmarking the
// ARM64 encoder without result allocation or the length-planning pass.
type EncodeSIMDBenchmarkPlan struct {
	plan encodePlan
}

// NewEncodeSIMDBenchmarkPlan prepares the state excluded from simd_core timing.
func NewEncodeSIMDBenchmarkPlan(s []rune) (EncodeSIMDBenchmarkPlan, int, bool) {
	plan := planEncodeSIMD(s)
	return EncodeSIMDBenchmarkPlan{plan: plan}, plan.size, true
}

// EncodeSIMDCoreForBenchmark runs only the hot encoder into caller-owned out.
// out must contain encodedSize+15 bytes for the encoder's padded final store.
func EncodeSIMDCoreForBenchmark(s []rune, out []byte, benchmarkPlan EncodeSIMDBenchmarkPlan) []byte {
	encodeSIMD(s, out, benchmarkPlan.plan)
	return out[:benchmarkPlan.plan.size]
}
