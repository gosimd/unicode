//go:build !goexperiment.simd || (!amd64 && !arm64)

package decode

import stdutf16 "unicode/utf16"

// Decode returns the rune sequence represented by s.
func Decode(s []uint16) []rune {
	return stdutf16.Decode(s)
}
