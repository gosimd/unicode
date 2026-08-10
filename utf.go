package utf

import simdutf8 "github.com/gosimd/unicode/simd/unicode/utf8"

// Valid reports whether p consists entirely of valid UTF-8.
func Valid(p []byte) bool {
	return simdutf8.Valid(p)
}

// ValidString reports whether s consists entirely of valid UTF-8.
func ValidString(s string) bool {
	return simdutf8.ValidString(s)
}

// RuneCount returns the number of runes in p. Erroneous and short encodings
// are treated as single runes of width 1 byte.
func RuneCount(p []byte) int {
	return simdutf8.RuneCount(p)
}
