//go:build goexperiment.simd && amd64

package utf16

import (
	"simd/archsimd"
	"unsafe"
)

// decodeAVX512 widens clean BMP chunks with AVX-512. A chunk containing a
// surrogate is decoded scalarly, preserving unicode/utf16.Decode semantics.
// The caller owns out, which must have room for len(s) runes.
func decodeAVX512(s []uint16, out []rune) []rune {
	if len(out) < len(s) {
		panic("utf16: output buffer too small")
	}

	mask := archsimd.BroadcastUint16x16(surrogateMask)
	marker := archsimd.BroadcastUint16x16(surrogateHighStart)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0
	for i < len(s) {
		if len(s)-i >= 2*decodeSIMDChunkSize {
			chunk := archsimd.LoadUint16x16Array((*[16]uint16)(unsafe.Add(inputBase, uintptr(i)*2)))
			surrogates := chunk.And(mask).Equal(marker)
			if surrogates.ToBits() == 0 {
				chunk.ExtendToUint32().BitsToInt32().StoreArray((*[16]int32)(unsafe.Add(outputBase, uintptr(n)*4)))
				i += 2 * decodeSIMDChunkSize
				n += 2 * decodeSIMDChunkSize
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
