//go:build !goexperiment.simd || (!amd64 && !arm64)

package encode

import stdutf16 "unicode/utf16"

// Encode returns the UTF-16 encoding of s.
func Encode(s []rune) []uint16 {
	return stdutf16.Encode(s)
}
