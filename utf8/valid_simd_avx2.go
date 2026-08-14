//go:build goexperiment.simd && amd64

package utf8

import "simd/archsimd"

const (
	avx2WindowSize = 512
	avx2VectorSize = 32
)

var (
	avx2SpecialPrevHighTable    = repeatAVX2NibbleTable(utf8SpecialPrevHighTable)
	avx2SpecialPrevLowTable     = repeatAVX2NibbleTable(utf8SpecialPrevLowTable)
	avx2SpecialCurrentHighTable = repeatAVX2NibbleTable(utf8SpecialCurrentHighTable)
)

func repeatAVX2NibbleTable(table [16]uint8) (wide [32]uint8) {
	copy(wide[0:16], table[:])
	copy(wide[16:32], table[:])
	return wide
}

// validAVX2 first skips wide, self-contained ASCII windows. A dirty window
// switches to the native 256-bit validator; short initial inputs and a final
// tail after clean ASCII windows retain the 128-bit baseline.
func validAVX2(p []byte) bool {
	highBit := archsimd.BroadcastUint8x32(0x80)
	for len(p) >= avx2WindowSize {
		if !allASCIIAVX2Block512(p, highBit) {
			return validAVX2NonASCII(p)
		}
		p = p[avx2WindowSize:]
	}
	return validAVX2Baseline(p)
}

type avx2UTF8State struct {
	previous   archsimd.Uint8x32
	tail       [3]byte
	incomplete bool
}

func (state avx2UTF8State) pending() bool {
	return state.incomplete
}

// validAVX2NonASCII starts at a 512-byte window known to contain non-ASCII
// bytes. Later independent ASCII windows can return to the cheap scan path.
func validAVX2NonASCII(p []byte) bool {
	var state avx2UTF8State
	if !validateAVX2DirtyBlock(p[:avx2WindowSize], &state) {
		return false
	}
	p = p[avx2WindowSize:]

	highBit := archsimd.BroadcastUint8x32(0x80)
	for len(p) >= avx2WindowSize {
		if !state.pending() && allASCIIAVX2Block512(p, highBit) {
			state = avx2UTF8State{}
			p = p[avx2WindowSize:]
			continue
		}
		if !validateAVX2DirtyBlock(p[:avx2WindowSize], &state) {
			return false
		}
		p = p[avx2WindowSize:]
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

// validateAVX2DirtyBlock validates sixteen 32-byte vectors and reduces their
// accumulated error bits once at the end of the window.
//
//go:noinline
func validateAVX2DirtyBlock(p []byte, state *avx2UTF8State) bool {
	lowNibbleMask := archsimd.BroadcastUint8x32(0x0f)
	continuationBit := archsimd.BroadcastUint8x32(0x80)
	needSecondContinuation := archsimd.BroadcastUint8x32(0x60)
	needThirdContinuation := archsimd.BroadcastUint8x32(0x70)
	zero := archsimd.Uint8x32{}

	prevHighTable := archsimd.LoadUint8x32Array(&avx2SpecialPrevHighTable)
	prevLowTable := archsimd.LoadUint8x32Array(&avx2SpecialPrevLowTable)
	currentHighTable := archsimd.LoadUint8x32Array(&avx2SpecialCurrentHighTable)

	previous := state.previous
	errors := zero
	blockStart := p
	for range avx2WindowSize / avx2VectorSize {
		block := archsimd.LoadUint8x32(p)

		// VPALIGNR works within 128-bit groups. Arrange the preceding groups
		// as [previous.high, block.low] so both groups receive adjacent bytes.
		previousGroups := previous.ConcatPermute128Scalars(1, 2, block)
		prev1 := block.ConcatShiftBytesRightGrouped(previousGroups, 15)
		prev2 := block.ConcatShiftBytesRightGrouped(previousGroups, 14)
		prev3 := block.ConcatShiftBytesRightGrouped(previousGroups, 13)

		prevHigh := prev1.AsUint16x16().ShiftAllRight(4).
			AsUint8x32().And(lowNibbleMask).AsInt8x32()
		prevLow := prev1.And(lowNibbleMask).AsInt8x32()
		currentHigh := block.AsUint16x16().ShiftAllRight(4).
			AsUint8x32().And(lowNibbleMask).AsInt8x32()
		specialCases := prevHighTable.PermuteOrZeroGrouped(prevHigh).
			And(prevLowTable.PermuteOrZeroGrouped(prevLow)).
			And(currentHighTable.PermuteOrZeroGrouped(currentHigh))

		mustContinue := prev2.SubSaturated(needSecondContinuation).Or(
			prev3.SubSaturated(needThirdContinuation),
		)
		errors = errors.Or(mustContinue.And(continuationBit).Xor(specialCases))

		previous = block
		p = p[avx2VectorSize:]
	}
	if !errors.IsZero() {
		return false
	}

	state.previous = previous
	state.tail[0], state.tail[1], state.tail[2] = blockStart[509], blockStart[510], blockStart[511]
	state.incomplete = state.tail[0] > 0xef || state.tail[1] > 0xdf || state.tail[2] > 0xbf
	return true
}

func allASCIIAVX2Block512(p []byte, highBit archsimd.Uint8x32) bool {
	acc0 := archsimd.LoadUint8x32(p).Or(
		archsimd.LoadUint8x32(p[32:])).Or(
		archsimd.LoadUint8x32(p[64:])).Or(
		archsimd.LoadUint8x32(p[96:]))
	acc1 := archsimd.LoadUint8x32(p[128:]).Or(
		archsimd.LoadUint8x32(p[160:])).Or(
		archsimd.LoadUint8x32(p[192:])).Or(
		archsimd.LoadUint8x32(p[224:]))
	acc2 := archsimd.LoadUint8x32(p[256:]).Or(
		archsimd.LoadUint8x32(p[288:])).Or(
		archsimd.LoadUint8x32(p[320:])).Or(
		archsimd.LoadUint8x32(p[352:]))
	acc3 := archsimd.LoadUint8x32(p[384:]).Or(
		archsimd.LoadUint8x32(p[416:])).Or(
		archsimd.LoadUint8x32(p[448:])).Or(
		archsimd.LoadUint8x32(p[480:]))
	return acc0.Or(acc1).Or(acc2.Or(acc3)).And(highBit).IsZero()
}

// validAVX2Baseline validates UTF-8 in 16-byte vectors. It only uses
// AVX/AVX2-width primitives, so AVX2-only CPUs never execute AVX-512.
func validAVX2Baseline(p []byte) bool {
	const offset1 = simdChunkSize
	const offset2 = 2 * simdChunkSize
	const offset3 = 3 * simdChunkSize

	prev := archsimd.Uint8x16{}
	incomplete := archsimd.Uint8x16{}
	for len(p) >= simdBlockSize {
		chunk0 := archsimd.LoadUint8x16(p)
		chunk1 := archsimd.LoadUint8x16(p[offset1:])
		chunk2 := archsimd.LoadUint8x16(p[offset2:])
		chunk3 := archsimd.LoadUint8x16(p[offset3:])

		if allASCIIBlock(chunk0, chunk1, chunk2, chunk3) {
			if !allZero(incomplete) {
				return false
			}
			prev = archsimd.Uint8x16{}
		} else {
			blockErrors := validateSIMDChunk(chunk0, prev)
			blockErrors = blockErrors.Or(validateSIMDChunk(chunk1, chunk0))
			blockErrors = blockErrors.Or(validateSIMDChunk(chunk2, chunk1))
			blockErrors = blockErrors.Or(validateSIMDChunk(chunk3, chunk2))
			prev = chunk3
			if !allZero(blockErrors) {
				return false
			}
			incomplete = incompleteSIMDChunk(chunk3)
		}
		p = p[simdBlockSize:]
	}

	var errors archsimd.Uint8x16
	for len(p) >= simdChunkSize {
		chunk := archsimd.LoadUint8x16(p)
		errors = errors.Or(validateSIMDChunk(chunk, prev))
		prev = chunk
		p = p[simdChunkSize:]
	}

	if !allZero(errors) {
		return false
	}

	incomplete = incompleteSIMDChunk(prev)
	if len(p) == 0 {
		return allZero(incomplete)
	}

	state, ok := stateForScalarTail(prev)
	if !ok {
		return false
	}

	for _, b := range p {
		if !state.step(b, classifyScalar(b)) {
			return false
		}
	}

	return state.complete()
}
