//go:build goexperiment.simd && amd64

package scan

import (
	"simd/archsimd"
	stdutf8 "unicode/utf8"
)

const (
	avx512WindowSize = 512
	avx512VectorSize = 64
)

var (
	avx512PreviousGroupsIndices = [8]uint64{6, 7, 8, 9, 10, 11, 12, 13}

	avx512SpecialPrevHighTable    = repeatAVX512NibbleTable(utf8SpecialPrevHighTable)
	avx512SpecialPrevLowTable     = repeatAVX512NibbleTable(utf8SpecialPrevLowTable)
	avx512SpecialCurrentHighTable = repeatAVX512NibbleTable(utf8SpecialCurrentHighTable)
)

func repeatAVX512NibbleTable(table [16]uint8) (wide [64]uint8) {
	copy(wide[0:16], table[:])
	copy(wide[16:32], table[:])
	copy(wide[32:48], table[:])
	copy(wide[48:64], table[:])
	return wide
}

func validAVX512(p []byte) bool {
	return validAVX512Native(p)
}

// validAVX512Native first accepts independent 512-byte ASCII blocks. A block
// containing non-ASCII bytes is validated in eight 64-byte AVX-512 vectors.
// Nibble lookups and saturating subtraction preserve UTF-8 dependencies
// across their boundaries without reducing masks to scalar registers.
func validAVX512Native(p []byte) bool {
	if len(p) < avx512WindowSize {
		return stdutf8.Valid(p)
	}

	highBit := archsimd.BroadcastUint8x64(0x80)
	zero := archsimd.Uint8x64{}

	var state avx512UTF8State
	for len(p) >= avx512WindowSize {
		// An ASCII block can be accepted independently, unless a sequence from
		// the preceding dirty block still needs continuation bytes.
		if !state.pending() && allASCIIBlock512(p, highBit, zero) {
			state = avx512UTF8State{}
			p = p[avx512WindowSize:]
			continue
		}
		if !validateAVX512DirtyBlock(p[:avx512WindowSize], &state) {
			return false
		}
		p = p[avx512WindowSize:]
	}

	scalarState, ok := stateForScalarTailBytes(state.tail[0], state.tail[1], state.tail[2])
	if !ok {
		return false
	}

	for _, b := range p {
		if !scalarState.step(b, classifyScalar(b)) {
			return false
		}
	}

	return scalarState.complete()
}

type avx512UTF8State struct {
	previous   archsimd.Uint8x64
	tail       [3]byte
	incomplete bool
}

func (state avx512UTF8State) pending() bool {
	return state.incomplete
}

// validateAVX512DirtyBlock validates one 512-byte block. Keeping the wide
// predicate constants here leaves the overwhelmingly common ASCII path small.
//
//go:noinline
func validateAVX512DirtyBlock(p []byte, state *avx512UTF8State) bool {
	lowNibbleMask := archsimd.BroadcastUint8x64(0x0f)
	continuationBit := archsimd.BroadcastUint8x64(0x80)
	needSecondContinuation := archsimd.BroadcastUint8x64(0x60)
	needThirdContinuation := archsimd.BroadcastUint8x64(0x70)
	zero := archsimd.Uint8x64{}

	previousGroupsIndices := archsimd.LoadUint64x8Array(&avx512PreviousGroupsIndices)
	prevHighTable := archsimd.LoadUint8x64Array(&avx512SpecialPrevHighTable)
	prevLowTable := archsimd.LoadUint8x64Array(&avx512SpecialPrevLowTable)
	currentHighTable := archsimd.LoadUint8x64Array(&avx512SpecialCurrentHighTable)

	previous := state.previous
	errors := zero
	for range avx512WindowSize / avx512VectorSize {
		block := archsimd.LoadUint8x64(p)

		// VPALIGNR operates independently in each 128-bit group. Arrange the
		// predecessor groups so every group receives the bytes immediately
		// preceding it, including the boundary from the previous ZMM vector.
		previousGroups := previous.AsUint64x8().ConcatPermute(
			block.AsUint64x8(), previousGroupsIndices,
		).AsUint8x64()
		prev1 := block.ConcatShiftBytesRightGrouped(previousGroups, 15)
		prev2 := block.ConcatShiftBytesRightGrouped(previousGroups, 14)
		prev3 := block.ConcatShiftBytesRightGrouped(previousGroups, 13)

		prevHigh := prev1.AsUint16x32().ShiftAllRight(4).
			AsUint8x64().And(lowNibbleMask).AsInt8x64()
		prevLow := prev1.And(lowNibbleMask).AsInt8x64()
		currentHigh := block.AsUint16x32().ShiftAllRight(4).
			AsUint8x64().And(lowNibbleMask).AsInt8x64()
		specialCases := prevHighTable.PermuteOrZeroGrouped(prevHigh).
			And(prevLowTable.PermuteOrZeroGrouped(prevLow)).
			And(currentHighTable.PermuteOrZeroGrouped(currentHigh))

		mustContinue := prev2.SubSaturated(needSecondContinuation).Or(
			prev3.SubSaturated(needThirdContinuation),
		)
		errors = errors.Or(mustContinue.And(continuationBit).Xor(specialCases))

		previous = block
		state.tail[0], state.tail[1], state.tail[2] = p[61], p[62], p[63]
		p = p[avx512VectorSize:]
	}
	if errors.Equal(zero).ToBits() != ^uint64(0) {
		return false
	}

	state.previous = previous
	state.incomplete = state.tail[0] > 0xef ||
		state.tail[1] > 0xdf ||
		state.tail[2] > 0xbf
	return true
}

func allASCIIBlock512(p []byte, highBit, zero archsimd.Uint8x64) bool {
	acc0 := archsimd.LoadUint8x64(p).Or(archsimd.LoadUint8x64(p[64:]))
	acc1 := archsimd.LoadUint8x64(p[128:]).Or(archsimd.LoadUint8x64(p[192:]))
	acc2 := archsimd.LoadUint8x64(p[256:]).Or(archsimd.LoadUint8x64(p[320:]))
	acc3 := archsimd.LoadUint8x64(p[384:]).Or(archsimd.LoadUint8x64(p[448:]))
	return acc0.Or(acc1).Or(acc2.Or(acc3)).And(highBit).Equal(zero).ToBits() == ^uint64(0)
}
