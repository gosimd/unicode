//go:build goexperiment.simd && (amd64 || arm64)

package utf8

import (
	"runtime"
	"simd/archsimd"
	stdutf8 "unicode/utf8"
)

const (
	simdChunkSize = 16
	simdBlockSize = 64
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

const (
	utf8ClassContinuation byte = 1 << iota
	utf8ClassLead2
	utf8ClassLead3
	utf8ClassLead4
)

var (
	utf8HighClassTable = [16]uint8{
		0, 0, 0, 0, 0, 0, 0, 0,
		utf8ClassContinuation, utf8ClassContinuation, utf8ClassContinuation, utf8ClassContinuation,
		utf8ClassLead2, utf8ClassLead2, utf8ClassLead3, utf8ClassLead4,
	}
	utf8InvalidC0C1LowTable = [16]uint8{
		0xff, 0xff, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}
	utf8ValidF0F4LowTable = [16]uint8{
		0xff, 0xff, 0xff, 0xff, 0xff, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0,
	}
	utf8InvalidF5FFLowTable = [16]uint8{
		0, 0, 0, 0, 0, 0xff, 0xff, 0xff,
		0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	}
)

const (
	utf8Invalid byte = iota
	utf8ASCII
	utf8Continuation
	utf8Lead2
	utf8Lead3
	utf8Lead4
)

type utf8State struct {
	need int
	lo   byte
	hi   byte
}

func (s *utf8State) complete() bool {
	return s.need == 0
}

func (s *utf8State) step(b byte, class byte) bool {
	if s.need != 0 {
		if class != utf8Continuation || b < s.lo || s.hi < b {
			return false
		}
		s.need--
		s.lo = 0x80
		s.hi = 0xbf
		return true
	}

	switch class {
	case utf8ASCII:
		return true
	case utf8Lead2:
		s.need = 1
		s.lo = 0x80
		s.hi = 0xbf
		return true
	case utf8Lead3:
		s.need = 2
		switch b {
		case 0xe0:
			s.lo = 0xa0
			s.hi = 0xbf
		case 0xed:
			s.lo = 0x80
			s.hi = 0x9f
		default:
			s.lo = 0x80
			s.hi = 0xbf
		}
		return true
	case utf8Lead4:
		s.need = 3
		switch b {
		case 0xf0:
			s.lo = 0x90
			s.hi = 0xbf
		case 0xf4:
			s.lo = 0x80
			s.hi = 0x8f
		default:
			s.lo = 0x80
			s.hi = 0xbf
		}
		return true
	default:
		return false
	}
}

// stateForScalarTail reconstructs the state needed to validate a final
// non-SIMD tail. Full chunks use incompleteSIMDChunk instead.
func stateForScalarTail(chunk archsimd.Uint8x16) (utf8State, bool) {
	b13 := chunk.GetElem(13)
	b14 := chunk.GetElem(14)
	b15 := chunk.GetElem(15)

	class15 := classifyScalar(b15)
	// The fused SIMD predicate validates a leading byte against its successor.
	// A malformed lead in the final lane has no successor in this chunk, so it
	// must be rejected while determining the state carried into the scalar tail.
	if class15 == utf8Invalid {
		return utf8State{}, false
	}
	if class15 == utf8Lead2 || class15 == utf8Lead3 || class15 == utf8Lead4 {
		var state utf8State
		if !state.step(b15, class15) {
			return utf8State{}, false
		}
		return state, true
	}

	if class := classifyScalar(b14); class == utf8Lead3 || class == utf8Lead4 {
		var state utf8State
		if !state.step(b14, class) || !state.step(b15, classifyScalar(b15)) {
			return utf8State{}, false
		}
		return state, true
	}

	if class := classifyScalar(b13); class == utf8Lead4 {
		var state utf8State
		if !state.step(b13, class) ||
			!state.step(b14, classifyScalar(b14)) ||
			!state.step(b15, classifyScalar(b15)) {
			return utf8State{}, false
		}
		return state, true
	}

	return utf8State{}, true
}

func classFlags(chunk archsimd.Uint8x16) archsimd.Uint8x16 {
	return lookupNibble(archsimd.LoadUint8x16Array(&utf8HighClassTable), highNibbles(chunk))
}

func hasClassFlag(flags archsimd.Uint8x16, flag byte) archsimd.Mask8x16 {
	return flags.And(archsimd.BroadcastUint8x16(flag)).NotEqual(archsimd.Uint8x16{})
}

func highNibbles(chunk archsimd.Uint8x16) archsimd.Uint8x16 {
	return chunk.ReshapeToUint16s().
		ShiftAllRight(4).
		ReshapeToUint8s().
		And(archsimd.BroadcastUint8x16(0x0f))
}

func lowNibbles(chunk archsimd.Uint8x16) archsimd.Uint8x16 {
	return chunk.And(archsimd.BroadcastUint8x16(0x0f))
}

func maskBits(mask archsimd.Mask8x16) archsimd.Uint8x16 {
	return mask.ToInt8x16().ToBits()
}

func classifyScalar(b byte) byte {
	switch {
	case b <= 0x7f:
		return utf8ASCII
	case 0x80 <= b && b <= 0xbf:
		return utf8Continuation
	case 0xc2 <= b && b <= 0xdf:
		return utf8Lead2
	case 0xe0 <= b && b <= 0xef:
		return utf8Lead3
	case 0xf0 <= b && b <= 0xf4:
		return utf8Lead4
	default:
		return utf8Invalid
	}
}
