//go:build goexperiment.simd && arm64

package decode

import "simd/archsimd"

// allASCIIBlock reports whether all 64 bytes are ASCII.
func allASCIIBlock(chunk0, chunk1, chunk2, chunk3 archsimd.Uint8x16) bool {
	acc := chunk0.Or(chunk1).Or(chunk2.Or(chunk3))
	return acc.BitsToInt8().ReduceMin() >= 0
}

func maskBits(mask archsimd.Mask8x16) archsimd.Uint8x16 {
	return mask.ToInt8x16().ToBits()
}
