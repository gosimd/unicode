//go:build goexperiment.simd && arm64

package utf8

import "simd/archsimd"

func allZero(chunk archsimd.Uint8x16) bool {
	return chunk.ReduceMax() == 0
}
