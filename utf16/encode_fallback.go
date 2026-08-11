//go:build !goexperiment.simd || (!amd64 && !arm64)

package utf16

import stdutf16 "unicode/utf16"

func encode(s []rune) []uint16 {
	return stdutf16.Encode(s)
}
