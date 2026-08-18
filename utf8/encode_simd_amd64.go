//go:build goexperiment.simd && amd64

package utf8

import (
	"simd/archsimd"
	"unsafe"
)

type encodePlan struct {
	size     int
	allASCII bool
	allValid bool
}

// Encode returns the UTF-8 encoding of s. Runes outside the valid Unicode
// range are replaced by RuneError, as in the language conversion string(s).
func Encode(s []rune) string {
	if len(s) == 0 {
		return ""
	}
	if !archsimd.X86.AVX2() {
		return string(s)
	}

	plan := planEncodeSIMD(s)
	// Both amd64 encoders may use a full 16-byte store for their final SIMD
	// group. The returned string excludes this padding.
	out := make([]byte, plan.size+15)
	encodeSIMD(s, out, plan)
	return unsafe.String(unsafe.SliceData(out), plan.size)
}

func planEncodeSIMD(s []rune) encodePlan {
	if archsimd.X86.AVX512() {
		return planEncodeAVX512(s)
	}
	return planEncodeAVX2(s)
}

func encodeSIMD(s []rune, out []byte, plan encodePlan) {
	if archsimd.X86.AVX512() {
		encodeAVX512(s, out, plan)
		return
	}
	encodeAVX2(s, out, plan)
}
