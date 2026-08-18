//go:build goexperiment.simd && amd64

package encode

import "simd/archsimd"

// EncodeSIMDBenchmarkPlan is an opaque test-only plan for the runtime-selected
// AVX2/AVX-512 encoder.
type EncodeSIMDBenchmarkPlan struct {
	plan      encodePlan
	available bool
}

func NewEncodeSIMDBenchmarkPlan(s []rune) (EncodeSIMDBenchmarkPlan, int, bool) {
	if !archsimd.X86.AVX2() {
		return EncodeSIMDBenchmarkPlan{}, 0, false
	}
	plan := planEncodeSIMD(s)
	return EncodeSIMDBenchmarkPlan{plan: plan, available: true}, plan.size, true
}

// EncodeSIMDCoreForBenchmark excludes planning and allocation.
func EncodeSIMDCoreForBenchmark(s []rune, out []byte, benchmarkPlan EncodeSIMDBenchmarkPlan) []byte {
	if !benchmarkPlan.available {
		panic("utf8: amd64 SIMD encoder is unavailable")
	}
	encodeSIMD(s, out, benchmarkPlan.plan)
	return out[:benchmarkPlan.plan.size]
}
