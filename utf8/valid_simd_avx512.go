//go:build goexperiment.simd && amd64

package utf8

import (
	"simd/archsimd"
	stdutf8 "unicode/utf8"
)

const (
	avx512WindowSize = 512
	avx512VectorSize = 64
)

func validAVX512(p []byte) bool {
	return validAVX512Native(p)
}

// validAVX512Native first accepts independent 512-byte ASCII blocks. A block
// containing non-ASCII bytes is validated in eight 64-byte AVX-512 vectors;
// scalar mask shifts preserve UTF-8 dependencies across their boundaries.
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
	need1, need2, lead4 uint64
	e0, ed, f0, f4      uint64
	tail                [3]byte
}

func (state avx512UTF8State) pending() bool {
	return ((state.need1>>63)|(state.need2>>62)|(state.lead4>>61))&0x7 != 0
}

// validateAVX512DirtyBlock validates one 512-byte block. Keeping the wide
// predicate constants here leaves the overwhelmingly common ASCII path small.
//
//go:noinline
func validateAVX512DirtyBlock(p []byte, state *avx512UTF8State) bool {
	c80 := archsimd.BroadcastUint8x64(0x80)
	c90 := archsimd.BroadcastUint8x64(0x90)
	c9f := archsimd.BroadcastUint8x64(0x9f)
	a0 := archsimd.BroadcastUint8x64(0xa0)
	c0 := archsimd.BroadcastUint8x64(0xc0)
	c2 := archsimd.BroadcastUint8x64(0xc2)
	c8f := archsimd.BroadcastUint8x64(0x8f)
	e0 := archsimd.BroadcastUint8x64(0xe0)
	ed := archsimd.BroadcastUint8x64(0xed)
	f0 := archsimd.BroadcastUint8x64(0xf0)
	f4 := archsimd.BroadcastUint8x64(0xf4)

	prevNeed1, prevNeed2, prevLead4 := state.need1, state.need2, state.lead4
	prevE0, prevED, prevF0, prevF4 := state.e0, state.ed, state.f0, state.f4
	for range avx512WindowSize / avx512VectorSize {
		block := archsimd.LoadUint8x64(p)

		continuation := block.GreaterEqual(c80).ToBits() & block.Less(c0).ToBits()
		lead2 := block.GreaterEqual(c2).ToBits() & block.Less(e0).ToBits()
		lead3 := block.GreaterEqual(e0).ToBits() & block.Less(f0).ToBits()
		lead4 := block.GreaterEqual(f0).ToBits() & block.LessEqual(f4).ToBits()
		need1 := lead2 | lead3 | lead4
		need2 := lead3 | lead4

		carry := ((prevNeed1 >> 63) & 1) |
			((prevNeed2 >> 62) & 1) |
			((prevLead4 >> 61) & 1)
		carry |= (((prevNeed2 >> 63) & 1) | ((prevLead4 >> 62) & 1)) << 1
		carry |= ((prevLead4 >> 63) & 1) << 2
		expected := (need1 << 1) | (need2 << 2) | (lead4 << 3) | carry

		errors := continuation ^ expected
		errors |= block.GreaterEqual(c0).ToBits() & block.Less(c2).ToBits()
		errors |= block.Greater(f4).ToBits()

		e0Mask := block.Equal(e0).ToBits()
		edMask := block.Equal(ed).ToBits()
		f0Mask := block.Equal(f0).ToBits()
		f4Mask := block.Equal(f4).ToBits()
		errors |= ((e0Mask << 1) | ((prevE0 >> 63) & 1)) & block.Less(a0).ToBits()
		errors |= ((edMask << 1) | ((prevED >> 63) & 1)) & block.Greater(c9f).ToBits()
		errors |= ((f0Mask << 1) | ((prevF0 >> 63) & 1)) & block.Less(c90).ToBits()
		errors |= ((f4Mask << 1) | ((prevF4 >> 63) & 1)) & block.Greater(c8f).ToBits()
		if errors != 0 {
			return false
		}

		prevNeed1, prevNeed2, prevLead4 = need1, need2, lead4
		prevE0, prevED, prevF0, prevF4 = e0Mask, edMask, f0Mask, f4Mask
		state.tail[0], state.tail[1], state.tail[2] = p[61], p[62], p[63]
		p = p[avx512VectorSize:]
	}
	state.need1, state.need2, state.lead4 = prevNeed1, prevNeed2, prevLead4
	state.e0, state.ed, state.f0, state.f4 = prevE0, prevED, prevF0, prevF4
	return true
}

func allASCIIBlock512(p []byte, highBit, zero archsimd.Uint8x64) bool {
	acc0 := archsimd.LoadUint8x64(p).Or(archsimd.LoadUint8x64(p[64:]))
	acc1 := archsimd.LoadUint8x64(p[128:]).Or(archsimd.LoadUint8x64(p[192:]))
	acc2 := archsimd.LoadUint8x64(p[256:]).Or(archsimd.LoadUint8x64(p[320:]))
	acc3 := archsimd.LoadUint8x64(p[384:]).Or(archsimd.LoadUint8x64(p[448:]))
	return acc0.Or(acc1).Or(acc2.Or(acc3)).And(highBit).Equal(zero).ToBits() == ^uint64(0)
}

// stateForScalarTailBytes reconstructs an incomplete sequence that can begin
// in the final three bytes of an already-valid SIMD block.
func stateForScalarTailBytes(b13, b14, b15 byte) (utf8State, bool) {
	class15 := classifyScalar(b15)
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
		if !state.step(b14, class) || !state.step(b15, class15) {
			return utf8State{}, false
		}
		return state, true
	}

	if class := classifyScalar(b13); class == utf8Lead4 {
		var state utf8State
		if !state.step(b13, class) ||
			!state.step(b14, classifyScalar(b14)) ||
			!state.step(b15, class15) {
			return utf8State{}, false
		}
		return state, true
	}

	return utf8State{}, true
}
