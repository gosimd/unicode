package utf8

import (
	internaldecode "github.com/gosimd/unicode/utf8/internal/decode"
	internalencode "github.com/gosimd/unicode/utf8/internal/encode"
	internalscan "github.com/gosimd/unicode/utf8/internal/scan"
)

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
