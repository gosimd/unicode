//go:build goexperiment.simd && amd64

package utf16

import "simd/archsimd"

// decodeSIMD widens clean BMP chunks with AVX2. A chunk containing a
// surrogate is decoded scalarly, preserving unicode/utf16.Decode semantics.
func decodeSIMD(s []uint16) []rune {
	out := make([]rune, len(s))
	mask := archsimd.BroadcastUint16x8(surrogateMask)
	marker := archsimd.BroadcastUint16x8(surrogateHighStart)
	i, n := 0, 0
	for i < len(s) {
		if len(s)-i >= decodeSIMDChunkSize {
			chunk := archsimd.LoadUint16x8(s[i:])
			surrogates := chunk.And(mask).Equal(marker)
			if surrogates.ToBits() == 0 {
				chunk.ExtendToUint32().BitsToInt32().Store(out[n:])
				i += decodeSIMDChunkSize
				n += decodeSIMDChunkSize
				continue
			}
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
