//go:build goexperiment.simd && amd64

package scan

import (
	"math/bits"
	"simd/archsimd"
)

func allASCIIBlock(chunk0, chunk1, chunk2, chunk3 archsimd.Uint8x16) bool {
	acc := chunk0.Or(chunk1).Or(chunk2.Or(chunk3))
	return allZero(acc.And(archsimd.BroadcastUint8x16(0x80)))
}

func continuationCount(chunk archsimd.Uint8x16) int {
	return bits.OnesCount16(continuationMask(chunk).ToBits())
}

func continuationCountBlock(chunk0, chunk1, chunk2, chunk3 archsimd.Uint8x16) int {
	masks := uint64(continuationMask(chunk0).ToBits()) |
		uint64(continuationMask(chunk1).ToBits())<<16 |
		uint64(continuationMask(chunk2).ToBits())<<32 |
		uint64(continuationMask(chunk3).ToBits())<<48
	return bits.OnesCount64(masks)
}

func highNibbles(chunk archsimd.Uint8x16) archsimd.Uint8x16 {
	return chunk.ReshapeToUint16s().
		ShiftAllRight(4).
		ReshapeToUint8s().
		And(archsimd.BroadcastUint8x16(0x0f))
}

func lookupNibble(table archsimd.Uint8x16, indices archsimd.Uint8x16) archsimd.Uint8x16 {
	return table.PermuteOrZero(indices.BitsToInt8())
}

func allZero(chunk archsimd.Uint8x16) bool {
	return chunk.IsZero()
}
