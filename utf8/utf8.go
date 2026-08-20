// Package utf8 provides SIMD-accelerated UTF-8 validation, rune counting, and
// conversion for Go.
//
// On supported ARM64 and AMD64 configurations, whole-buffer operations use
// SIMD to provide substantial speedups over the Go standard library. In
// representative 64 KiB benchmarks, selected workloads achieve 5x to 25x
// speedups. Results depend on the operation, input shape, CPU, and build
// configuration; unsupported configurations use a portable fallback with the
// same behaviour.
package utf8

import (
	internaldecode "github.com/gosimd/unicode/utf8/internal/decode"
	internalencode "github.com/gosimd/unicode/utf8/internal/encode"
	internalscan "github.com/gosimd/unicode/utf8/internal/scan"
	stdutf8 "unicode/utf8"
)

const (
	RuneError = stdutf8.RuneError
	RuneSelf  = stdutf8.RuneSelf
	MaxRune   = stdutf8.MaxRune
	UTFMax    = stdutf8.UTFMax
)

// AppendRune appends the UTF-8 encoding of r to the end of p and returns the
// extended buffer. If the rune is out of range, it appends the encoding of
// RuneError.
func AppendRune(p []byte, r rune) []byte {
	return stdutf8.AppendRune(p, r)
}

// DecodeLastRune unpacks the last UTF-8 encoding in p and returns the rune and
// its width in bytes.
func DecodeLastRune(p []byte) (r rune, size int) {
	return stdutf8.DecodeLastRune(p)
}

// DecodeLastRuneInString is like DecodeLastRune but its input is a string.
func DecodeLastRuneInString(s string) (r rune, size int) {
	return stdutf8.DecodeLastRuneInString(s)
}

// DecodeRune unpacks the first UTF-8 encoding in p and returns the rune and its
// width in bytes.
func DecodeRune(p []byte) (r rune, size int) {
	return stdutf8.DecodeRune(p)
}

// DecodeRuneInString is like DecodeRune but its input is a string.
func DecodeRuneInString(s string) (r rune, size int) {
	return stdutf8.DecodeRuneInString(s)
}

// EncodeRune writes into p the UTF-8 encoding of r and returns the number of
// bytes written. If the rune is out of range, it writes the encoding of
// RuneError.
func EncodeRune(p []byte, r rune) int {
	return stdutf8.EncodeRune(p, r)
}

// FullRune reports whether the bytes in p begin with a full UTF-8 encoding of a
// rune.
func FullRune(p []byte) bool {
	return stdutf8.FullRune(p)
}

// FullRuneInString is like FullRune but its input is a string.
func FullRuneInString(s string) bool {
	return stdutf8.FullRuneInString(s)
}

// RuneLen returns the number of bytes required to encode the rune.
func RuneLen(r rune) int {
	return stdutf8.RuneLen(r)
}

// RuneStart reports whether the byte could be the first byte of an encoded
// rune.
func RuneStart(b byte) bool {
	return stdutf8.RuneStart(b)
}

// ValidRune reports whether r can be legally encoded as UTF-8.
func ValidRune(r rune) bool {
	return stdutf8.ValidRune(r)
}

// Valid reports whether p consists entirely of valid UTF-8.
func Valid(p []byte) bool {
	return internalscan.Valid(p)
}

// ValidString reports whether s consists entirely of valid UTF-8.
func ValidString(s string) bool {
	return internalscan.ValidString(s)
}

// RuneCount returns the number of runes in p. Erroneous and short encodings
// are treated as single runes of width 1 byte.
func RuneCount(p []byte) int {
	return internalscan.RuneCount(p)
}

// RuneCountInString is like RuneCount but its input is a string.
func RuneCountInString(s string) int {
	return internalscan.RuneCountInString(s)
}

// Encode returns the UTF-8 encoding of s. Runes outside the valid Unicode
// range are replaced by RuneError, as in the language conversion string(s).
func Encode(s []rune) string {
	return internalencode.Encode(s)
}

// Decode returns the runes in s. Invalid UTF-8 encodings are replaced by
// RuneError, as in the language conversion []rune(s).
func Decode(s string) []rune {
	return internaldecode.Decode(s)
}
