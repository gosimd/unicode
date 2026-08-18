//go:build goexperiment.simd && arm64

package encode

func AvailableForBenchmark() bool {
	return true
}

// EncodeCoreForBenchmark measures planning and SIMD encoding without output
// allocation. out must have room for twice as many code units as s has runes.
func EncodeCoreForBenchmark(s []rune, out []uint16) []uint16 {
	plan := planEncodeSIMD(s)
	return encodeSIMDWithPlan(s, out, plan)
}
