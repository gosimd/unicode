//go:build goexperiment.simd && amd64

package utf8

import (
	"math/bits"
	"simd/archsimd"
)

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
