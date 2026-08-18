//go:build !goexperiment.simd || (!arm64 && !amd64)

package decode

// Decode returns the runes in s. Invalid UTF-8 encodings are replaced by
// RuneError, as in the language conversion []rune(s).
func Decode(s string) []rune {
	return []rune(s)
}
