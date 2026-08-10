//go:build goexperiment.simd && arm64

package utf8

import "simd/archsimd"

// allASCIIBlock reports whether all 64 bytes are ASCII. Reinterpreting the
// combined bytes as signed makes every non-ASCII byte negative, so a signed
// minimum avoids both the 0x80 broadcast and the mask operation.
func allASCIIBlock(chunk0, chunk1, chunk2, chunk3 archsimd.Uint8x16) bool {
	acc := chunk0.Or(chunk1).Or(chunk2.Or(chunk3))
	return acc.BitsToInt8().ReduceMin() >= 0
}
