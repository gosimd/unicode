//go:build goexperiment.simd && amd64

package utf8

import "simd/archsimd"

func allZero(chunk archsimd.Uint8x16) bool {
	return chunk.IsZero()
}
