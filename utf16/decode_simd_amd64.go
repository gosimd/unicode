//go:build goexperiment.simd && amd64

package utf16

import "simd/archsimd"

// decodeSIMDChunk decodes eight non-surrogate code units using AVX2. It
// returns false without writing when the chunk needs scalar surrogate logic.
func decodeSIMDChunk(s []uint16, out []rune) bool {
	chunk := archsimd.LoadUint16x8(s)
	surrogates := chunk.GreaterEqual(archsimd.BroadcastUint16x8(surrogateHighStart)).And(
		chunk.Less(archsimd.BroadcastUint16x8(surrogateEnd)),
	)
	if surrogates.ToBits() != 0 {
		return false
	}

	chunk.ExtendToUint32().BitsToInt32().Store(out)
	return true
}
