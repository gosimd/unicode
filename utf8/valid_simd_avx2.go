//go:build goexperiment.simd && amd64

package utf8

import "simd/archsimd"

const avx2ASCIIBlockSize = 512

// validAVX2 first skips wide, self-contained ASCII blocks. The first
// non-ASCII byte is left for validAVX2Baseline, which preserves the full
// cross-chunk UTF-8 state machine.
func validAVX2(p []byte) bool {
	highBit := archsimd.BroadcastUint8x32(0x80)
	for len(p) >= avx2ASCIIBlockSize {
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
		if !acc0.Or(acc1).Or(acc2.Or(acc3)).And(highBit).IsZero() {
			break
		}
		p = p[avx2ASCIIBlockSize:]
	}
	return validAVX2Baseline(p)
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
