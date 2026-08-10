//go:build !goexperiment.simd || (!amd64 && !arm64)

package utf16

import stdutf16 "unicode/utf16"

func decode(s []uint16) []rune {
	return stdutf16.Decode(s)
}
