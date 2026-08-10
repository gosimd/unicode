//go:build goexperiment.simd && amd64

package utf8

import "simd/archsimd"

func allASCIIBlock(chunk0, chunk1, chunk2, chunk3 archsimd.Uint8x16) bool {
	acc := chunk0.Or(chunk1).Or(chunk2.Or(chunk3))
	return allZero(acc.And(archsimd.BroadcastUint8x16(0x80)))
}
