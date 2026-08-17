//go:build goexperiment.simd && amd64

package utf16

import (
	"math/bits"
	"simd/archsimd"
	"unsafe"
)

var (
	decodeSurrogatePairBaseAVX2 = [16]uint16{
		surrogateHighStart, surrogateLowStart, surrogateHighStart, surrogateLowStart,
		surrogateHighStart, surrogateLowStart, surrogateHighStart, surrogateLowStart,
		surrogateHighStart, surrogateLowStart, surrogateHighStart, surrogateLowStart,
		surrogateHighStart, surrogateLowStart, surrogateHighStart, surrogateLowStart,
	}
	decodeSurrogatePairWeightsAVX2 = [16]int16{
		1024, 1, 1024, 1, 1024, 1, 1024, 1,
		1024, 1, 1024, 1, 1024, 1, 1024, 1,
	}
	decodeSparsePairPermuteAVX2 = [7][8]uint32{
		{0, 2, 3, 4, 5, 6, 7, 7},
		{0, 1, 3, 4, 5, 6, 7, 7},
		{0, 1, 2, 4, 5, 6, 7, 7},
		{0, 1, 2, 3, 5, 6, 7, 7},
		{0, 1, 2, 3, 4, 6, 7, 7},
		{0, 1, 2, 3, 4, 5, 7, 7},
		{0, 1, 2, 3, 4, 5, 6, 7},
	}
)

// decodeAVX2 widens clean BMP chunks with AVX2.
func decodeAVX2(s []uint16, out []rune) []rune {
	if len(out) < len(s) {
		panic("utf16: output buffer too small")
	}

	mask := archsimd.BroadcastUint16x8(surrogateMask)
	marker := archsimd.BroadcastUint16x8(surrogateHighStart)
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
			noSurrogates := chunk0.And(mask).Equal(marker).
				Or(chunk1.And(mask).Equal(marker)).
				Or(chunk2.And(mask).Equal(marker)).
				Or(chunk3.And(mask).Equal(marker)).
				ToInt16x8().IsZero()
			if noSurrogates {
				chunk0.ExtendToUint32().BitsToInt32().StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n)*4)))
				chunk1.ExtendToUint32().BitsToInt32().StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n+8)*4)))
				chunk2.ExtendToUint32().BitsToInt32().StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n+16)*4)))
				chunk3.ExtendToUint32().BitsToInt32().StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n+24)*4)))
				i += 4 * decodeSIMDChunkSize
				n += 4 * decodeSIMDChunkSize
				continue
			}

			i, n = decodeSparseRunAVX2(s, out, i, n)
			continue
		}

		if len(s)-i >= decodeSIMDChunkSize {
			chunk := archsimd.LoadUint16x8Array((*[8]uint16)(unsafe.Add(inputBase, uintptr(i)*2)))
			noSurrogates := chunk.And(mask).Equal(marker).
				ToInt16x8().IsZero()
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

//go:noinline
func decodeSparseRunAVX2(s []uint16, out []rune, i, n int) (int, int) {
	mask := archsimd.BroadcastUint16x8(surrogateMask)
	marker := archsimd.BroadcastUint16x8(surrogateHighStart)
	zero := archsimd.BroadcastInt8x16(0)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	cleanChunks := 0

	for len(s)-i >= decodeSIMDChunkSize {
		chunk := archsimd.LoadUint16x8Array((*[8]uint16)(unsafe.Add(inputBase, uintptr(i)*2)))
		surrogates := chunk.And(mask).Equal(marker)
		if surrogates.ToInt16x8().IsZero() {
			chunk.ExtendToUint32().BitsToInt32().StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n)*4)))
			i += decodeSIMDChunkSize
			n += decodeSIMDChunkSize
			cleanChunks++
			if cleanChunks == 4 {
				return i, n
			}
			continue
		}

		cleanChunks = 0
		surrogateBits := ^surrogates.ToInt16x8().AsUint8x16().BitsToInt8().Equal(zero).ToBits()
		bit := bits.TrailingZeros16(surrogateBits)
		pairLane := bit / 2
		// A pair beginning in the final lane consumes the first code unit
		// after the chunk but still produces eight output runes.
		if pairLane == decodeSIMDChunkSize-1 && surrogateBits == 0xC000 && len(s)-i > decodeSIMDChunkSize {
			high := *(*uint16)(unsafe.Add(inputBase, uintptr(i+pairLane)*2))
			low := *(*uint16)(unsafe.Add(inputBase, uintptr(i+decodeSIMDChunkSize)*2))
			if high-surrogateHighStart < surrogateLowStart-surrogateHighStart &&
				low-surrogateLowStart < surrogateEnd-surrogateLowStart {
				chunk.ExtendToUint32().BitsToInt32().
					StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n)*4)))
				decoded := (rune(high-surrogateHighStart) << 10) |
					rune(low-surrogateLowStart) + surrogateOffset
				*(*int32)(unsafe.Add(outputBase, uintptr(n+pairLane)*4)) = int32(decoded)
				i += decodeSIMDChunkSize + 1
				n += decodeSIMDChunkSize
				continue
			}
		}
		if pairLane < decodeSIMDChunkSize-1 && surrogateBits == uint16(0xF)<<bit {
			high := *(*uint16)(unsafe.Add(inputBase, uintptr(i+pairLane)*2))
			low := *(*uint16)(unsafe.Add(inputBase, uintptr(i+pairLane+1)*2))
			if high-surrogateHighStart < surrogateLowStart-surrogateHighStart &&
				low-surrogateLowStart < surrogateEnd-surrogateLowStart {
				indices := archsimd.LoadUint32x8Array(&decodeSparsePairPermuteAVX2[pairLane])
				// The eighth stored lane is outside the logical result and is
				// overwritten by the next chunk.
				chunk.ExtendToUint32().Permute(indices).BitsToInt32().
					StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n)*4)))
				decoded := (rune(high-surrogateHighStart) << 10) |
					rune(low-surrogateLowStart) + surrogateOffset
				*(*int32)(unsafe.Add(outputBase, uintptr(n+pairLane)*4)) = int32(decoded)
				i += decodeSIMDChunkSize
				n += decodeSIMDChunkSize - 1
				continue
			}
		}

		end := i + decodeSIMDChunkSize
		scalarN := n
		i, n = decodeScalar(s, out, i, n, end)
		if n-scalarN == decodeSIMDChunkSize/2 && len(s)-i >= 4*decodeSIMDChunkSize {
			i, n = decodeSurrogatePairRunAVX2(inputBase, outputBase, len(s), i, n)
		}
	}
	return i, n
}

//go:noinline
func decodeSurrogatePairRunAVX2(inputBase, outputBase unsafe.Pointer, inputLen, i, n int) (int, int) {
	base := archsimd.LoadUint16x16Array(&decodeSurrogatePairBaseAVX2)
	maximum := archsimd.BroadcastUint16x16(surrogateLowStart - surrogateHighStart - 1)
	weights := archsimd.LoadInt16x16Array(&decodeSurrogatePairWeightsAVX2)
	offset := archsimd.BroadcastInt32x8(surrogateOffset)

	for inputLen-i >= 4*decodeSIMDChunkSize {
		chunk0 := archsimd.LoadUint16x16Array((*[16]uint16)(unsafe.Add(inputBase, uintptr(i)*2)))
		chunk1 := archsimd.LoadUint16x16Array((*[16]uint16)(unsafe.Add(inputBase, uintptr(i+16)*2)))
		diff0 := chunk0.Sub(base)
		diff1 := chunk1.Sub(base)
		bad := diff0.SubSaturated(maximum).Or(diff1.SubSaturated(maximum))
		if !bad.IsZero() {
			break
		}

		diff0.AsInt16x16().DotProductPairs(weights).Add(offset).
			StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n)*4)))
		diff1.AsInt16x16().DotProductPairs(weights).Add(offset).
			StoreArray((*[8]int32)(unsafe.Add(outputBase, uintptr(n+8)*4)))
		i += 4 * decodeSIMDChunkSize
		n += 2 * decodeSIMDChunkSize
	}
	return i, n
}
