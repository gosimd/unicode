//go:build goexperiment.simd && arm64

package scan

import "simd/archsimd"

func validSIMD(p []byte) bool {
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
