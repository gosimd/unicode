//go:build goexperiment.simd && arm64

package utf8

import "simd/archsimd"

const (
	utf8TooShort         = 1 << 0
	utf8TooLong          = 1 << 1
	utf8Overlong3        = 1 << 2
	utf8TooLarge         = 1 << 3
	utf8Surrogate        = 1 << 4
	utf8Overlong2        = 1 << 5
	utf8TooLarge1000     = 1 << 6
	utf8TwoContinuations = 1 << 7

	utf8Carry = utf8TooShort | utf8TooLong | utf8TwoContinuations
)

var (
	// These tables fuse leading-byte validity, continuation validity, and the
	// E0, ED, F0, and F4 boundary rules into a single error-bit vector.
	utf8SpecialPrevHighTable = [16]uint8{
		utf8TooLong, utf8TooLong, utf8TooLong, utf8TooLong,
		utf8TooLong, utf8TooLong, utf8TooLong, utf8TooLong,
		utf8TwoContinuations, utf8TwoContinuations, utf8TwoContinuations, utf8TwoContinuations,
		utf8TooShort | utf8Overlong2,
		utf8TooShort,
		utf8TooShort | utf8Overlong3 | utf8Surrogate,
		utf8TooShort | utf8TooLarge | utf8TooLarge1000,
	}
	utf8SpecialPrevLowTable = [16]uint8{
		utf8Carry | utf8Overlong3 | utf8Overlong2 | utf8TooLarge1000,
		utf8Carry | utf8Overlong2,
		utf8Carry, utf8Carry,
		utf8Carry | utf8TooLarge,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000 | utf8Surrogate,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
		utf8Carry | utf8TooLarge | utf8TooLarge1000,
	}
	utf8SpecialCurrentHighTable = [16]uint8{
		utf8TooShort, utf8TooShort, utf8TooShort, utf8TooShort,
		utf8TooShort, utf8TooShort, utf8TooShort, utf8TooShort,
		utf8TooLong | utf8Overlong2 | utf8TwoContinuations | utf8Overlong3 | utf8TooLarge1000,
		utf8TooLong | utf8Overlong2 | utf8TwoContinuations | utf8Overlong3 | utf8TooLarge,
		utf8TooLong | utf8Overlong2 | utf8TwoContinuations | utf8Surrogate | utf8TooLarge,
		utf8TooLong | utf8Overlong2 | utf8TwoContinuations | utf8Surrogate | utf8TooLarge,
		utf8TooShort, utf8TooShort, utf8TooShort, utf8TooShort,
	}
)

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
