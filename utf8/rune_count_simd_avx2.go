//go:build goexperiment.simd && amd64

package utf8

import "simd/archsimd"

// runeCountAVX2 validates and counts full 512-byte windows with AVX2.
// Inputs shorter than one window retain the 128-bit implementation.
func runeCountAVX2(p []byte) (int, bool) {
	if len(p) < avx2WindowSize {
		return runeCountSIMD128(p)
	}

	highBit := archsimd.BroadcastUint8x32(0x80)

	count := 0
	var state avx2UTF8State
	var continuationSums archsimd.Uint64x4
	for len(p) >= avx2WindowSize {
		if !state.pending() && allASCIIAVX2Block512(p, highBit) {
			state = avx2UTF8State{}
			count += avx2WindowSize
			p = p[avx2WindowSize:]
			continue
		}

		if !validateAndCountAVX2DirtyBlock(
			p[:avx2WindowSize], &state, &continuationSums,
		) {
			return 0, false
		}
		count += avx2WindowSize
		p = p[avx2WindowSize:]
	}

	var sums [4]uint64
	continuationSums.StoreArray(&sums)
	continuationFlags := sums[0] + sums[1] + sums[2] + sums[3]
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

// validateAndCountAVX2DirtyBlock validates one 512-byte block and adds the
// byte sums of its continuation-class flags to continuationSums. An invalid
// block discards the partial count by returning false to the scalar fallback.
//
//go:noinline
func validateAndCountAVX2DirtyBlock(
	p []byte,
	state *avx2UTF8State,
	continuationSums *archsimd.Uint64x4,
) bool {
	lowNibbleMask := archsimd.BroadcastUint8x32(0x0f)
	continuationBit := archsimd.BroadcastUint8x32(0x80)
	continuationClassBit := archsimd.BroadcastUint8x32(utf8TooLong)
	needSecondContinuation := archsimd.BroadcastUint8x32(0x60)
	needThirdContinuation := archsimd.BroadcastUint8x32(0x70)

	prevHighTable := archsimd.LoadUint8x32Array(&avx2SpecialPrevHighTable)
	prevLowTable := archsimd.LoadUint8x32Array(&avx2SpecialPrevLowTable)
	currentHighTable := archsimd.LoadUint8x32Array(&avx2SpecialCurrentHighTable)

	previous := state.previous
	var errors archsimd.Uint8x32
	sums := *continuationSums
	blockStart := p
	for range avx2WindowSize / avx2VectorSize {
		block := archsimd.LoadUint8x32(p)

		previousGroups := previous.ConcatPermute128Scalars(1, 2, block)
		prev1 := block.ConcatShiftBytesRightGrouped(previousGroups, 15)
		prev2 := block.ConcatShiftBytesRightGrouped(previousGroups, 14)
		prev3 := block.ConcatShiftBytesRightGrouped(previousGroups, 13)

		prevHigh := prev1.AsUint16x16().ShiftAllRight(4).
			AsUint8x32().And(lowNibbleMask).AsInt8x32()
		prevLow := prev1.And(lowNibbleMask).AsInt8x32()
		currentHigh := block.AsUint16x16().ShiftAllRight(4).
			AsUint8x32().And(lowNibbleMask).AsInt8x32()
		currentFlags := currentHighTable.PermuteOrZeroGrouped(currentHigh)
		specialCases := prevHighTable.PermuteOrZeroGrouped(prevHigh).
			And(prevLowTable.PermuteOrZeroGrouped(prevLow)).
			And(currentFlags)

		mustContinue := prev2.SubSaturated(needSecondContinuation).Or(
			prev3.SubSaturated(needThirdContinuation),
		)
		expected := mustContinue.And(continuationBit)
		continuations := currentFlags.And(continuationClassBit)
		// errors stays zero for valid input. Once it is non-zero, the partial
		// continuation sum is discarded with the failed block.
		sums = sums.Add(continuations.SumOf8AbsDiff(errors))
		errors = errors.Or(expected.Xor(specialCases))

		previous = block
		p = p[avx2VectorSize:]
	}
	if !errors.IsZero() {
		return false
	}

	state.previous = previous
	state.tail[0], state.tail[1], state.tail[2] = blockStart[509], blockStart[510], blockStart[511]
	state.incomplete = state.tail[0] > 0xef ||
		state.tail[1] > 0xdf ||
		state.tail[2] > 0xbf
	*continuationSums = sums
	return true
}
