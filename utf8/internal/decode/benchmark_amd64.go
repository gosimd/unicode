//go:build goexperiment.simd && amd64

package decode

import (
	"github.com/gosimd/unicode/utf8/internal/scan"
	"simd/archsimd"
	"unsafe"
)

// DecodeSIMDBenchmarkPlan is an opaque, test-only plan for benchmarking the
// runtime-selected AVX2/AVX-512 decoder without validation, rune counting, or
// result allocation.
type DecodeSIMDBenchmarkPlan struct {
	runes     int
	available bool
}

// NewDecodeSIMDBenchmarkPlan prepares the state excluded from simd_core timing.
// Invalid input deliberately has no core plan: decodeValidSIMD accepts only
// already-validated UTF-8.
func NewDecodeSIMDBenchmarkPlan(s string) (DecodeSIMDBenchmarkPlan, int, bool) {
	if !archsimd.X86.AVX2() {
		return DecodeSIMDBenchmarkPlan{}, 0, false
	}
	input := unsafe.Slice(unsafe.StringData(s), len(s))
	count, valid := scan.CountValid(input)
	return DecodeSIMDBenchmarkPlan{runes: count, available: valid}, count, valid
}

// DecodeSIMDCoreForBenchmark runs only the hot decoder into caller-owned out.
// out must contain decodedRunes writable lanes.
func DecodeSIMDCoreForBenchmark(s string, out []rune, plan DecodeSIMDBenchmarkPlan) []rune {
	if !plan.available {
		panic("utf8: amd64 SIMD decoder is unavailable")
	}
	input := unsafe.Slice(unsafe.StringData(s), len(s))
	return decodeValidSIMD(input, out, plan.runes)
}
