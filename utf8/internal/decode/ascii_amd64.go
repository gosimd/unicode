//go:build goexperiment.simd && amd64

package decode

import "simd/archsimd"

func allASCIIBlock(chunk0, chunk1, chunk2, chunk3 archsimd.Uint8x16) bool {
	acc := chunk0.Or(chunk1).Or(chunk2.Or(chunk3))
	return acc.And(archsimd.BroadcastUint8x16(0x80)).IsZero()
}

func continuationMask(chunk archsimd.Uint8x16) archsimd.Mask8x16 {
	return chunk.And(archsimd.BroadcastUint8x16(0xc0)).Equal(archsimd.BroadcastUint8x16(0x80))
}
