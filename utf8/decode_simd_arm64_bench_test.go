//go:build goexperiment.simd && arm64

package utf8

import "unsafe"

// DecodeSIMDBenchmarkPlan is an opaque, test-only plan for benchmarking the
// ARM64 decoder without validation, rune counting, or result allocation.
type DecodeSIMDBenchmarkPlan struct {
	runes int
}

// NewDecodeSIMDBenchmarkPlan prepares the state excluded from simd_core timing.
// Invalid input deliberately has no core plan: decodeValidSIMD accepts only
// already-validated UTF-8.
func NewDecodeSIMDBenchmarkPlan(s string) (DecodeSIMDBenchmarkPlan, int, bool) {
	input := unsafe.Slice(unsafe.StringData(s), len(s))
	count, valid := runeCountSIMD(input)
	return DecodeSIMDBenchmarkPlan{runes: count}, count, valid
}

// DecodeSIMDCoreForBenchmark runs only the hot decoder into caller-owned out.
// out must contain decodedRunes writable lanes.
func DecodeSIMDCoreForBenchmark(s string, out []rune, plan DecodeSIMDBenchmarkPlan) []rune {
	input := unsafe.Slice(unsafe.StringData(s), len(s))
	return decodeValidSIMD(input, out, plan.runes)
}
