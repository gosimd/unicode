//go:build goexperiment.simd && arm64

package utf8

import "simd/archsimd"

// incompleteSIMDChunk marks leading bytes in the final three lanes that do
// not have enough following lanes in this chunk. SubSaturated yields zero for
// all complete positions and a non-zero carry marker otherwise.
func incompleteSIMDChunk(chunk archsimd.Uint8x16) archsimd.Uint8x16 {
	return chunk.SubSaturated(archsimd.LoadUint8x16Array(&utf8IncompleteThresholds))
}

// validateSIMDChunk validates one 16-byte chunk using the algorithm described
// by simdutf. It represents UTF-8 constraints as error bits, so the first
// continuation byte and all exceptional leading-byte cases share three table
// lookups instead of separate masks and comparisons.
func validateSIMDChunk(chunk archsimd.Uint8x16, prev archsimd.Uint8x16) archsimd.Uint8x16 {
	prev1 := chunk.ConcatShiftBytesRight(prev, 15)
	special := specialCaseErrors(chunk, prev1)

	prev2 := chunk.ConcatShiftBytesRight(prev, 14)
	prev3 := chunk.ConcatShiftBytesRight(prev, 13)
	mustContinue := prev2.SubSaturated(archsimd.BroadcastUint8x16(0x60)).Or(
		prev3.SubSaturated(archsimd.BroadcastUint8x16(0x70)),
	)

	return mustContinue.And(archsimd.BroadcastUint8x16(0x80)).Xor(special)
}

func specialCaseErrors(chunk archsimd.Uint8x16, prev1 archsimd.Uint8x16) archsimd.Uint8x16 {
	prevHigh := lookupNibble(
		archsimd.LoadUint8x16Array(&utf8SpecialPrevHighTable),
		highNibbles(prev1),
	)
	prevLow := lookupNibble(
		archsimd.LoadUint8x16Array(&utf8SpecialPrevLowTable),
		lowNibbles(prev1),
	)
	currentHigh := lookupNibble(
		archsimd.LoadUint8x16Array(&utf8SpecialCurrentHighTable),
		highNibbles(chunk),
	)
	return prevHigh.And(prevLow).And(currentHigh)
}
