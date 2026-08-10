//go:build goexperiment.simd && amd64

package utf8

import "simd/archsimd"

func highNibbles(chunk archsimd.Uint8x16) archsimd.Uint8x16 {
	return chunk.ReshapeToUint16s().
		ShiftAllRight(4).
		ReshapeToUint8s().
		And(archsimd.BroadcastUint8x16(0x0f))
}
