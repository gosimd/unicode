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
		if len(s)-i >= 4*decodeSIMDChunkSize {
			// Test four independent chunks with one AVX2 mask extraction. This
			// reduces branch and address-generation overhead for clean BMP text.
			chunk0 := archsimd.LoadUint16x8Array((*[8]uint16)(unsafe.Add(inputBase, uintptr(i)*2)))
			chunk1 := archsimd.LoadUint16x8Array((*[8]uint16)(unsafe.Add(inputBase, uintptr(i+8)*2)))
			chunk2 := archsimd.LoadUint16x8Array((*[8]uint16)(unsafe.Add(inputBase, uintptr(i+16)*2)))
			chunk3 := archsimd.LoadUint16x8Array((*[8]uint16)(unsafe.Add(inputBase, uintptr(i+24)*2)))
			// Mask16x8.ToBits uses AVX-512 on amd64. Reinterpreting the
			// comparison as bytes lets Mask8x16.ToBits emit AVX VPMOVMSKB.
			noSurrogates := chunk0.And(mask).Equal(marker).
				Or(chunk1.And(mask).Equal(marker)).
				Or(chunk2.And(mask).Equal(marker)).
				Or(chunk3.And(mask).Equal(marker)).
				ToInt16x8().AsUint8x16().BitsToInt8().Equal(zero).ToBits() == ^uint16(0)
			if noSurrogates {
				chunk0.ExtendToUint32().BitsToInt32().StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n)*4)))
				chunk1.ExtendToUint32().BitsToInt32().StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n+8)*4)))
				chunk2.ExtendToUint32().BitsToInt32().StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n+16)*4)))
				chunk3.ExtendToUint32().BitsToInt32().StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n+24)*4)))
				i += 4 * decodeSIMDChunkSize
				n += 4 * decodeSIMDChunkSize
				continue
			}
		}

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
