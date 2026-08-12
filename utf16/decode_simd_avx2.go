//go:build goexperiment.simd && amd64

package utf16

import (
	"simd/archsimd"
	"unsafe"
)

// decodeAVX2 widens clean BMP chunks with AVX2. Its surrogate predicate uses
// a byte-mask VPMOVMSKB extraction, so it is safe on AVX2-only CPUs.
func decodeAVX2(s []uint16, out []rune) []rune {
	if len(out) < len(s) {
		panic("utf16: output buffer too small")
	}

	mask := archsimd.BroadcastUint16x8(surrogateMask)
	marker := archsimd.BroadcastUint16x8(surrogateHighStart)
	zero := archsimd.BroadcastInt8x16(0)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0
	for i < len(s) {
		if len(s)-i >= decodeSIMDChunkSize {
			chunk := archsimd.LoadUint16x8Array((*[8]uint16)(unsafe.Add(inputBase, uintptr(i)*2)))
			// Mask16x8.ToBits uses AVX-512 on amd64. Reinterpreting the
			// comparison as bytes lets Mask8x16.ToBits emit AVX VPMOVMSKB.
			noSurrogates := chunk.And(mask).Equal(marker).
				ToInt16x8().AsUint8x16().BitsToInt8().Equal(zero).ToBits() == ^uint16(0)
			if noSurrogates {
				chunk.ExtendToUint32().BitsToInt32().StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n)*4)))
				i += decodeSIMDChunkSize
				n += decodeSIMDChunkSize
				continue
			}
		}

		end := i + decodeSIMDChunkSize
		if end > len(s) {
			end = len(s)
		}
		i, n = decodeScalar(s, out, i, n, end)
	}
	return out[:n]
}
