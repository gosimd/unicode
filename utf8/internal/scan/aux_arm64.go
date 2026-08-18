//go:build goexperiment.simd && arm64

package scan

import "simd/archsimd"

// allASCIIBlock reports whether all 64 bytes are ASCII. Reinterpreting the
// combined bytes as signed makes every non-ASCII byte negative, so a signed
// minimum avoids both the 0x80 broadcast and the mask operation.
func allASCIIBlock(chunk0, chunk1, chunk2, chunk3 archsimd.Uint8x16) bool {
	acc := chunk0.Or(chunk1).Or(chunk2.Or(chunk3))
	return acc.BitsToInt8().ReduceMin() >= 0
}

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

func highNibbles(chunk archsimd.Uint8x16) archsimd.Uint8x16 {
	return chunk.ShiftAllRight(4)
}

func lookupNibble(table archsimd.Uint8x16, indices archsimd.Uint8x16) archsimd.Uint8x16 {
	return table.LookupOrZero(indices)
}

func allZero(chunk archsimd.Uint8x16) bool {
	return chunk.ReduceMax() == 0
}
