//go:build goexperiment.simd && arm64

package utf8

import "simd/archsimd"

func continuationCount(chunk archsimd.Uint8x16) int {
	return int(continuationOnes(chunk).ReduceSum())
}

func continuationCountBlock(chunk0, chunk1, chunk2, chunk3 archsimd.Uint8x16) int {
	ones := continuationOnes(chunk0).
		Add(continuationOnes(chunk1)).
		Add(continuationOnes(chunk2)).
		Add(continuationOnes(chunk3))
	return int(ones.ReduceSum())
}

func continuationOnes(chunk archsimd.Uint8x16) archsimd.Uint8x16 {
	return maskBits(continuationMask(chunk)).ShiftAllRight(7)
}
