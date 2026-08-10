//go:build goexperiment.simd && arm64

package utf16

import "simd/archsimd"

// decodeSIMDChunk decodes eight non-surrogate code units using NEON. It
// returns false without writing when the chunk needs scalar surrogate logic.
func decodeSIMDChunk(s []uint16, out []rune) bool {
	chunk := archsimd.LoadUint16x8(s)
	surrogates := chunk.GreaterEqual(archsimd.BroadcastUint16x8(surrogateHighStart)).And(
		chunk.Less(archsimd.BroadcastUint16x8(surrogateEnd)),
	)
	if surrogates.ToInt16x8().ToBits().ReduceMax() != 0 {
		return false
	}

	chunk.ExtendLo4ToUint32().BitsToInt32().Store(out)
	chunk.HiToLo().ExtendLo4ToUint32().BitsToInt32().Store(out[4:])
	return true
}
