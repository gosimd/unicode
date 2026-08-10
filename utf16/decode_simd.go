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
	replacementRune     = '\uFFFD'
	surrogateOffset     = 0x10000
)

func decode(s []uint16) []rune {
	if runtime.GOARCH == "amd64" && !archsimd.X86.AVX2() {
		return stdutf16.Decode(s)
	}

	return decodeSIMD(s)
}

// decodeSIMD widens clean BMP chunks with SIMD. Any chunk containing a
// surrogate is decoded one code unit at a time, which preserves the standard
// library's handling of pairs and malformed UTF-16.
func decodeSIMD(s []uint16) []rune {
	out := make([]rune, len(s))
	i, n := 0, 0
	for i < len(s) {
		if len(s)-i >= decodeSIMDChunkSize && decodeSIMDChunk(s[i:], out[n:]) {
			i += decodeSIMDChunkSize
			n += decodeSIMDChunkSize
			continue
		}

		// Decode an entire rejected chunk scalarly before testing the next one.
		// Testing after every scalar rune regresses inputs with dense surrogates.
		end := i + decodeSIMDChunkSize
		if end > len(s) {
			end = len(s)
		}
		i, n = decodeScalar(s, out, i, n, end)
	}
	return out[:n]
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
