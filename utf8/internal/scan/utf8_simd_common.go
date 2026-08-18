//go:build goexperiment.simd && (amd64 || arm64)

package scan

import "simd/archsimd"

const (
	simdChunkSize = 16
	simdBlockSize = 64
)

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
	return stateForScalarTailBytes(b13, b14, b15)
}

// stateForScalarTailBytes reconstructs an incomplete sequence that can begin
// in the final three bytes of an already-valid SIMD block.
func stateForScalarTailBytes(b13, b14, b15 byte) (utf8State, bool) {
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

func lowNibbles(chunk archsimd.Uint8x16) archsimd.Uint8x16 {
	return chunk.And(archsimd.BroadcastUint8x16(0x0f))
}

func maskBits(mask archsimd.Mask8x16) archsimd.Uint8x16 {
	return mask.ToInt8x16().ToBits()
}

func continuationMask(chunk archsimd.Uint8x16) archsimd.Mask8x16 {
	return chunk.And(archsimd.BroadcastUint8x16(0xc0)).Equal(archsimd.BroadcastUint8x16(0x80))
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
