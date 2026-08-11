//go:build goexperiment.simd && (amd64 || arm64)

package utf16

import (
	"runtime"
	"simd/archsimd"
	stdutf16 "unicode/utf16"
)

const (
	decodeSIMDChunkSize = 8
	surrogateHighStart  = 0xD800
	surrogateLowStart   = 0xDC00
	surrogateEnd        = 0xE000
	surrogateMask       = 0xF800
	replacementRune     = '\uFFFD'
	surrogateOffset     = 0x10000
)

func decode(s []uint16) []rune {
	if runtime.GOARCH == "amd64" && !archsimd.X86.AVX2() {
		return stdutf16.Decode(s)
	}

	out := make([]rune, len(s))
	return decodeSIMD(s, out)
}

// decodeScalar decodes s[i:end] with the same replacement and surrogate-pair
// rules as unicode/utf16.Decode. A pair may consume the first code unit after
// end when its high surrogate is the final code unit in the range.
func decodeScalar(s []uint16, out []rune, i, n, end int) (int, int) {
	for i < end {
		r := s[i]
		switch {
		case r < surrogateHighStart || surrogateEnd <= r:
			out[n] = rune(r)
			i++
		case r < surrogateLowStart && i+1 < len(s) &&
			surrogateLowStart <= s[i+1] && s[i+1] < surrogateEnd:
			out[n] = (rune(r-surrogateHighStart) << 10) |
				rune(s[i+1]-surrogateLowStart) + surrogateOffset
			i += 2
		default:
			out[n] = replacementRune
			i++
		}
		n++
	}
	return i, n
}
