//go:build goexperiment.simd && wasm

package utf8

import "simd/archsimd"

func allZero(chunk archsimd.Uint8x16) bool {
	bits := chunk.ReshapeToUint64s()
	return bits.GetElem(0)|bits.GetElem(1) == 0
}
