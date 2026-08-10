//go:build goexperiment.simd && (amd64 || arm64)

package utf8

import (
	"runtime"
	"simd/archsimd"
	stdutf8 "unicode/utf8"
	"unsafe"
)

// RuneCount returns the number of runes in p. Erroneous and short encodings
// are treated as single runes of width 1 byte.
func RuneCount(p []byte) int {
	if runtime.GOARCH == "amd64" && !archsimd.X86.AVX2() {
		return stdutf8.RuneCount(p)
	}

	n, ok := runeCountSIMD(p)
	if !ok {
		return stdutf8.RuneCount(p)
	}
	return n
}

// RuneCountInString is like RuneCount but its input is a string.
func RuneCountInString(s string) int {
	if len(s) == 0 {
		return 0
	}
	return RuneCount(unsafe.Slice(unsafe.StringData(s), len(s)))
}

// runeCountSIMD validates and counts p in one pass. On malformed input it
// returns false so RuneCount can preserve unicode/utf8's width-1 error-rune
// semantics with its scalar fallback.
func runeCountSIMD(p []byte) (int, bool) {
	p = p[:len(p):len(p)]
	const offset1 = simdChunkSize
	const offset2 = 2 * simdChunkSize
	const offset3 = 3 * simdChunkSize

	count := 0
	prev := archsimd.Uint8x16{}
	incomplete := archsimd.Uint8x16{}
	for len(p) >= simdBlockSize {
		chunk0 := archsimd.LoadUint8x16(p)
		chunk1 := archsimd.LoadUint8x16(p[offset1:])
		chunk2 := archsimd.LoadUint8x16(p[offset2:])
		chunk3 := archsimd.LoadUint8x16(p[offset3:])

		if allASCIIBlock(chunk0, chunk1, chunk2, chunk3) {
			if !allZero(incomplete) {
				return 0, false
			}
			count += simdBlockSize
			prev = archsimd.Uint8x16{}
		} else {
			errors := validateSIMDChunk(chunk0, prev)
			errors = errors.Or(validateSIMDChunk(chunk1, chunk0))
			errors = errors.Or(validateSIMDChunk(chunk2, chunk1))
			errors = errors.Or(validateSIMDChunk(chunk3, chunk2))
			if !allZero(errors) {
				return 0, false
			}
			count += simdBlockSize - continuationCountBlock(chunk0, chunk1, chunk2, chunk3)
			prev = chunk3
			incomplete = incompleteSIMDChunk(chunk3)
		}
		p = p[simdBlockSize:]
	}

	var errors archsimd.Uint8x16
	for len(p) >= simdChunkSize {
		chunk := archsimd.LoadUint8x16(p)
		errors = errors.Or(validateSIMDChunk(chunk, prev))
		count += simdChunkSize - continuationCount(chunk)
		prev = chunk
		p = p[simdChunkSize:]
	}
	if !allZero(errors) {
		return 0, false
	}

	incomplete = incompleteSIMDChunk(prev)
	if len(p) == 0 {
		return count, allZero(incomplete)
	}

	state, ok := stateForScalarTail(prev)
	if !ok {
		return 0, false
	}
	for _, b := range p {
		class := classifyScalar(b)
		if !state.step(b, class) {
			return 0, false
		}
		if class != utf8Continuation {
			count++
		}
	}
	return count, state.complete()
}
