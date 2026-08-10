//go:build goexperiment.simd && (amd64 || arm64)

package utf8

import (
	"runtime"
	"simd/archsimd"
	stdutf8 "unicode/utf8"
	"unsafe"
)

// Valid reports whether p consists entirely of valid UTF-8.
func Valid(p []byte) bool {
	if runtime.GOARCH == "amd64" && !archsimd.X86.AVX2() {
		return stdutf8.Valid(p)
	}

	// This optimization avoids the need to recompute the capacity
	// when generating code for slicing p, bringing it to parity with
	// ValidString, which was 20% faster on long ASCII strings.
	p = p[:len(p):len(p)]
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

// ValidString reports whether s consists entirely of valid UTF-8.
//
// It presents s to Valid without copying. Valid only reads its input, so the
// resulting byte slice does not violate string immutability.
func ValidString(s string) bool {
	if len(s) == 0 {
		return true
	}
	return Valid(unsafe.Slice(unsafe.StringData(s), len(s)))
}
