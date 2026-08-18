//go:build goexperiment.simd && amd64

package scan

import "simd/archsimd"

// runeCountAVX512 validates and counts full 512-byte windows with AVX-512.
// Inputs shorter than one window retain the existing 128-bit implementation.
func runeCountAVX512(p []byte) (int, bool) {
	if len(p) < avx512WindowSize {
		return runeCountSIMD128(p)
	}

	highBit := archsimd.BroadcastUint8x64(0x80)
	zero := archsimd.Uint8x64{}

	count := 0
	var state avx512UTF8State
	var continuationSums archsimd.Uint64x8
	for len(p) >= avx512WindowSize {
		if !state.pending() && allASCIIBlock512(p, highBit, zero) {
			state = avx512UTF8State{}
			count += avx512WindowSize
			p = p[avx512WindowSize:]
			continue
		}

		if !validateAndCountAVX512DirtyBlock(
			p[:avx512WindowSize], &state, &continuationSums,
		) {
			return 0, false
		}
		count += avx512WindowSize
		p = p[avx512WindowSize:]
	}

	var sums [8]uint64
	continuationSums.StoreArray(&sums)
	continuationFlags := sums[0] + sums[1] + sums[2] + sums[3] +
		sums[4] + sums[5] + sums[6] + sums[7]
	count -= int(continuationFlags >> 1)

	scalarState, ok := stateForScalarTailBytes(state.tail[0], state.tail[1], state.tail[2])
	if !ok {
		return 0, false
	}
	for _, b := range p {
		class := classifyScalar(b)
		if !scalarState.step(b, class) {
			return 0, false
		}
		if class != utf8Continuation {
			count++
		}
	}

	return count, scalarState.complete()
}

// validateAndCountAVX512DirtyBlock validates one 512-byte block and adds the
// byte sums of its continuation-class flags to continuationSums. An invalid
// block discards the partial count by returning false to the scalar fallback.
//
//go:noinline
func validateAndCountAVX512DirtyBlock(
	p []byte,
	state *avx512UTF8State,
	continuationSums *archsimd.Uint64x8,
) bool {
	lowNibbleMask := archsimd.BroadcastUint8x64(0x0f)
	continuationBit := archsimd.BroadcastUint8x64(0x80)
	continuationClassBit := archsimd.BroadcastUint8x64(utf8TooLong)
	needSecondContinuation := archsimd.BroadcastUint8x64(0x60)
	needThirdContinuation := archsimd.BroadcastUint8x64(0x70)
	zero := archsimd.Uint8x64{}

	previousGroupsIndices := archsimd.LoadUint64x8Array(&avx512PreviousGroupsIndices)
	prevHighTable := archsimd.LoadUint8x64Array(&avx512SpecialPrevHighTable)
	prevLowTable := archsimd.LoadUint8x64Array(&avx512SpecialPrevLowTable)
	currentHighTable := archsimd.LoadUint8x64Array(&avx512SpecialCurrentHighTable)

	previous := state.previous
	errors := zero
	sums := *continuationSums
	for range avx512WindowSize / avx512VectorSize {
		block := archsimd.LoadUint8x64(p)

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
		currentFlags := currentHighTable.PermuteOrZeroGrouped(currentHigh)
		specialCases := prevHighTable.PermuteOrZeroGrouped(prevHigh).
			And(prevLowTable.PermuteOrZeroGrouped(prevLow)).
			And(currentFlags)

		mustContinue := prev2.SubSaturated(needSecondContinuation).Or(
			prev3.SubSaturated(needThirdContinuation),
		)
		expected := mustContinue.And(continuationBit)
		errors = errors.Or(expected.Xor(specialCases))
		continuations := currentFlags.And(continuationClassBit)
		sums = sums.Add(continuations.SumOf8AbsDiff(zero))

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
	*continuationSums = sums
	return true
}
