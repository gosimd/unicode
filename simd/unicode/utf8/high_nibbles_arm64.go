//go:build goexperiment.simd && arm64

package utf8

import "simd/archsimd"

func highNibbles(chunk archsimd.Uint8x16) archsimd.Uint8x16 {
	return chunk.ShiftAllRight(4)
}
