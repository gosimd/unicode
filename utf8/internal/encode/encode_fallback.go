//go:build !goexperiment.simd || (!arm64 && !amd64)

package encode

// Encode returns the UTF-8 encoding of s. Runes outside the valid Unicode
// range are replaced by RuneError, as in the language conversion string(s).
func Encode(s []rune) string {
	return string(s)
}
