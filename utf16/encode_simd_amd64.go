//go:build goexperiment.simd && amd64

package utf16

import (
	"simd/archsimd"
	stdutf16 "unicode/utf16"
)

// encode selects AVX-512 when available, then AVX2. All other hosts use the
// standard library implementation.
func encode(s []rune) []uint16 {
	if !archsimd.X86.AVX2() {
		return stdutf16.Encode(s)
	}

	plan := planEncodeSIMD(s)
	out := make([]uint16, plan.capacity)
	return encodeSIMDWithPlan(s, out, plan)
}

// encodeSIMD encodes s into out. out must have room for the maximum encoded
// length, including an additional code unit for every rune at or above U+10000.
func encodeSIMD(s []rune, out []uint16, outputCapacity int) []uint16 {
	plan := planEncodeSIMD(s)
	return encodeSIMDWithPlan(s, out, plan)
}

func planEncodeSIMD(s []rune) encodingPlan {
	if archsimd.X86.AVX512() {
		capacity, mode := encodedLengthAVX512Profile(s)
		return encodingPlan{capacity: capacity, mode: uint8(mode)}
	}
	capacity, mode := encodedLengthAVX2Profile(s)
	return encodingPlan{capacity: capacity, mode: uint8(mode)}
}

func encodeSIMDWithPlan(s []rune, out []uint16, plan encodingPlan) []uint16 {
	if archsimd.X86.AVX512() {
		return encodeAVX512(s, out, plan.capacity, encodeAVX512Mode(plan.mode))
	}
	return encodeAVX2(s, out, plan.capacity, encodeAVX2Mode(plan.mode))
}

func encodedLengthSIMD(s []rune) int {
	return planEncodeSIMD(s).capacity
}
