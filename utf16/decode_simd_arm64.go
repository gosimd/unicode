//go:build goexperiment.simd && arm64

package utf16

import (
	"simd/archsimd"
	"unsafe"
)

// decodeSIMD widens clean BMP chunks with NEON. A chunk containing a
// surrogate is decoded scalarly, preserving unicode/utf16.Decode semantics.
// The caller owns out, which must have room for len(s) runes.
func decodeSIMD(s []uint16, out []rune) []rune {
	if len(out) < len(s) {
		panic("utf16: output buffer too small")
	}

	mask := archsimd.BroadcastUint16x8(surrogateMask)
	marker := archsimd.BroadcastUint16x8(surrogateHighStart)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0
	for i < len(s) {
		if len(s)-i >= decodeSIMDChunkSize {
			// The loop condition proves that this load and both stores fit.
			// Using array SIMD primitives avoids slice bounds checks here.
			chunk := archsimd.LoadUint16x8Array((*[8]uint16)(unsafe.Add(inputBase, uintptr(i)*2)))
			surrogates := chunk.And(mask).Equal(marker)
			if surrogates.ToInt16x8().ReduceMin() >= 0 {
				chunk.ExtendLo4ToUint32().BitsToInt32().StoreArray((*[4]int32)(unsafe.Add(outputBase, uintptr(n)*4)))
				chunk.HiToLo().ExtendLo4ToUint32().BitsToInt32().StoreArray((*[4]int32)(unsafe.Add(outputBase, uintptr(n+4)*4)))
				i += decodeSIMDChunkSize
				n += decodeSIMDChunkSize
				continue
			}
		}

		// Decode an entire rejected chunk scalarly before testing the next one.
		// Testing after every scalar rune regresses inputs with dense surrogates.
		end := i + decodeSIMDChunkSize
		if end > len(s) {
			end = len(s)
		}
		i, n = decodeScalar(s, out, i, n, end)
	}
	return out[:n]
}
