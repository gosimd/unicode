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
