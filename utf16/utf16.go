// Package utf16 provides SIMD-accelerated UTF-16 encoding and decoding for Go.
//
// It is API-compatible with the Go standard library's unicode/utf16 package.
// Whole-buffer Encode and Decode use SIMD on supported ARM64 and AMD64
// configurations, giving substantial speedups on representative workloads.
// Results depend on the input shape, CPU, and build configuration; unsupported
// configurations use the standard-library implementation with the same
// behaviour.
package utf16

import (
	internaldecode "github.com/gosimd/unicode/utf16/internal/decode"
	internalencode "github.com/gosimd/unicode/utf16/internal/encode"
	stdutf16 "unicode/utf16"
)

// IsSurrogate reports whether r can appear in a UTF-16 surrogate pair.
func IsSurrogate(r rune) bool {
	return stdutf16.IsSurrogate(r)
}

// DecodeRune returns the rune encoded by the UTF-16 surrogate pair r1, r2.
// It returns U+FFFD if r1 and r2 are not a valid surrogate pair.
func DecodeRune(r1, r2 rune) rune {
	return stdutf16.DecodeRune(r1, r2)
}

// EncodeRune returns the UTF-16 surrogate pair encoding r.
// It returns U+FFFD, U+FFFD if r is not a valid code point needing encoding.
func EncodeRune(r rune) (r1, r2 rune) {
	return stdutf16.EncodeRune(r)
}

// RuneLen returns the number of UTF-16 code units needed to encode r.
// It returns -1 if r is not a valid code point to encode in UTF-16.
func RuneLen(r rune) int {
	return stdutf16.RuneLen(r)
}

// Encode returns the UTF-16 encoding of s.
func Encode(s []rune) []uint16 {
	return internalencode.Encode(s)
}

// AppendRune appends the UTF-16 encoding of r to a and returns the result.
func AppendRune(a []uint16, r rune) []uint16 {
	return stdutf16.AppendRune(a, r)
}

// Decode returns the rune sequence represented by the UTF-16 code units s.
func Decode(s []uint16) []rune {
	return internaldecode.Decode(s)
}
