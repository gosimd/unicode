//go:build goexperiment.simd && amd64

package utf16

import (
	"math/bits"
	"simd/archsimd"
	"unsafe"
)

const encodeSIMDChunkSize = 4

type encodeAVX2Mode uint8

const (
	encodeAVX2Scalar encodeAVX2Mode = iota
	encodeAVX2Width8
	encodeAVX2Width16
	encodeAVX2LowBMP
)

var encodeAVX2PackOrder = [8]uint32{0, 1, 4, 5, 2, 3, 6, 7}

// encodeAVX2 dispatches to the density profile selected by the length pass.
// Its BMP packing uses only AVX2 instructions.
func encodeAVX2(s []rune, out []uint16, outputCapacity int, mode encodeAVX2Mode) []uint16 {
	if len(out) < outputCapacity {
		panic("utf16: output buffer too small")
	}
	if mode == encodeAVX2Scalar {
		_, n := encodeScalar(s, out, 0, 0, len(s))
		return out[:n]
	}
	if mode == encodeAVX2Width8 {
		return encodeAVX2Short(s, out, outputCapacity)
	}
	if mode == encodeAVX2LowBMP {
		return encodeAVX2LowBMPOnly(s, out, outputCapacity)
	}
	return encodeAVX2Wide(s, out, outputCapacity)
}

func encodeAVX2Wide(s []rune, out []uint16, outputCapacity int) []uint16 {
	highBits := archsimd.BroadcastUint32x8(^uint32(0xFFFF))
	surrogateBits := archsimd.BroadcastUint32x8(surrogateMask)
	surrogateMarker := archsimd.BroadcastUint32x8(surrogateHighStart)
	packOrder := archsimd.LoadUint32x8Array(&encodeAVX2PackOrder)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0
	for i < len(s) {
		if len(s)-i >= 4*encodeSIMDChunkSize {
			chunk0 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
			chunk1 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+8)*4)))
			invalid := invalidBMPAVX2Wide(chunk0, highBits, surrogateBits, surrogateMarker).
				Or(invalidBMPAVX2Wide(chunk1, highBits, surrogateBits, surrogateMarker))
			if invalid.IsZero() {
				packBMPAVX2(chunk0, chunk1, packOrder).
					StoreArray((*[16]uint16)(unsafe.Add(outputBase, uintptr(n)*2)))
				i += 4 * encodeSIMDChunkSize
				n += 4 * encodeSIMDChunkSize
				continue
			}
		}

		end := i + 2*encodeSIMDChunkSize
		if end > len(s) {
			end = len(s)
		}
		i, n = encodeScalar(s, out, i, n, end)
	}
	return out[:n]
}

func invalidBMPAVX2Wide(chunk, highBits, surrogateBits, surrogateMarker archsimd.Uint32x8) archsimd.Uint32x8 {
	return chunk.And(highBits).Or(
		chunk.And(surrogateBits).Equal(surrogateMarker).ToInt32x8().AsUint32x8(),
	)
}

func packBMPAVX2(chunk0, chunk1, order archsimd.Uint32x8) archsimd.Uint16x16 {
	return chunk0.BitsToInt32().SaturateToUint16ConcatGrouped(chunk1.BitsToInt32()).
		AsUint32x8().Permute(order).AsUint16x16()
}

// encodeAVX2Short is the original 8-rune AVX2 loop. AVX-512 uses it when the
// length pass observes clean 8-rune windows but not clean 16-rune windows.
func encodeAVX2Short(s []rune, out []uint16, outputCapacity int) []uint16 {
	if len(out) < outputCapacity {
		panic("utf16: output buffer too small")
	}

	highBits := archsimd.BroadcastUint32x4(^uint32(0xFFFF))
	surrogateBits := archsimd.BroadcastUint32x4(surrogateMask)
	surrogateMarker := archsimd.BroadcastUint32x4(surrogateHighStart)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0
	for i < len(s) {
		if len(s)-i >= 2*encodeSIMDChunkSize {
			chunk0 := archsimd.LoadUint32x4Array((*[4]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
			chunk1 := archsimd.LoadUint32x4Array((*[4]uint32)(unsafe.Add(inputBase, uintptr(i+encodeSIMDChunkSize)*4)))
			invalid := invalidBMPAVX2(chunk0, highBits, surrogateBits, surrogateMarker).
				Or(invalidBMPAVX2(chunk1, highBits, surrogateBits, surrogateMarker))
			if invalid.IsZero() {
				packed := chunk0.BitsToInt32().SaturateToUint16Concat(chunk1.BitsToInt32())
				packed.StoreArray((*[8]uint16)(unsafe.Add(outputBase, uintptr(n)*2)))
				i += 2 * encodeSIMDChunkSize
				n += 2 * encodeSIMDChunkSize
				continue
			}
		}

		end := i + 2*encodeSIMDChunkSize
		if end > len(s) {
			end = len(s)
		}
		i, n = encodeScalar(s, out, i, n, end)
	}
	return out[:n]
}

func encodedLengthAVX2(s []rune) int {
	capacity, _ := encodedLengthAVX2Profile(s)
	return capacity
}

// encodedLengthAVX2Profile first looks for the common low-BMP case, where the
// encoder can omit all value checks. Otherwise it counts non-BMP runes and
// chooses a width from their density.
func encodedLengthAVX2Profile(s []rune) (int, encodeAVX2Mode) {
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	lowBMPThreshold := archsimd.BroadcastUint32x8(surrogateHighStart)
	i := 0
	lowBMP := true
	for ; len(s)-i >= 8*encodeSIMDChunkSize; i += 8 * encodeSIMDChunkSize {
		chunk0 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
		chunk1 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+8)*4)))
		chunk2 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+16)*4)))
		chunk3 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+24)*4)))
		maximum := chunk0.Max(chunk1).Max(chunk2.Max(chunk3))
		if maximum.Less(lowBMPThreshold).ToBits() != 0xFF {
			lowBMP = false
			break
		}
	}
	if lowBMP {
		for ; i < len(s) && uint32(s[i]) < surrogateHighStart; i++ {
		}
		if i == len(s) {
			return len(s), encodeAVX2LowBMP
		}
	}

	n := len(s)
	threshold := archsimd.BroadcastInt32x8(surrogateOffset)
	for ; len(s)-i >= 8*encodeSIMDChunkSize; i += 8 * encodeSIMDChunkSize {
		chunk0 := archsimd.LoadInt32x8Array((*[8]int32)(unsafe.Add(inputBase, uintptr(i)*4)))
		chunk1 := archsimd.LoadInt32x8Array((*[8]int32)(unsafe.Add(inputBase, uintptr(i+8)*4)))
		chunk2 := archsimd.LoadInt32x8Array((*[8]int32)(unsafe.Add(inputBase, uintptr(i+16)*4)))
		chunk3 := archsimd.LoadInt32x8Array((*[8]int32)(unsafe.Add(inputBase, uintptr(i+24)*4)))
		mask0 := chunk0.GreaterEqual(threshold)
		mask1 := chunk1.GreaterEqual(threshold)
		mask2 := chunk2.GreaterEqual(threshold)
		mask3 := chunk3.GreaterEqual(threshold)
		if !mask0.Or(mask1).Or(mask2.Or(mask3)).ToInt32x8().IsZero() {
			n += bits.OnesCount8(mask0.ToBits()) + bits.OnesCount8(mask1.ToBits()) +
				bits.OnesCount8(mask2.ToBits()) + bits.OnesCount8(mask3.ToBits())
		}
	}
	for ; len(s)-i >= 2*encodeSIMDChunkSize; i += 2 * encodeSIMDChunkSize {
		chunk := archsimd.LoadInt32x8Array((*[8]int32)(unsafe.Add(inputBase, uintptr(i)*4)))
		n += bits.OnesCount8(chunk.GreaterEqual(threshold).ToBits())
	}
	for ; i < len(s); i++ {
		if s[i] >= surrogateOffset {
			n++
		}
	}

	extra := n - len(s)
	switch {
	case extra == 0 || extra*32 <= len(s):
		return n, encodeAVX2Width16
	case extra*8 <= len(s):
		return n, encodeAVX2Width8
	default:
		return n, encodeAVX2Scalar
	}
}

func invalidBMPAVX2(chunk, highBits, surrogateBits, surrogateMarker archsimd.Uint32x4) archsimd.Uint32x4 {
	return chunk.And(highBits).Or(
		chunk.And(surrogateBits).Equal(surrogateMarker).ToInt32x4().AsUint32x4(),
	)
}

//go:noinline
func encodeAVX2LowBMPOnly(s []rune, out []uint16, outputCapacity int) []uint16 {
	if len(out) < outputCapacity {
		panic("utf16: output buffer too small")
	}

	order := archsimd.LoadUint32x8Array(&encodeAVX2PackOrder)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i := 0
	for ; len(s)-i >= 16*encodeSIMDChunkSize; i += 16 * encodeSIMDChunkSize {
		chunk0 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
		chunk1 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+8)*4)))
		chunk2 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+16)*4)))
		chunk3 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+24)*4)))
		chunk4 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+32)*4)))
		chunk5 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+40)*4)))
		chunk6 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+48)*4)))
		chunk7 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+56)*4)))
		packBMPAVX2(chunk0, chunk1, order).StoreArray((*[16]uint16)(unsafe.Add(outputBase, uintptr(i)*2)))
		packBMPAVX2(chunk2, chunk3, order).StoreArray((*[16]uint16)(unsafe.Add(outputBase, uintptr(i+16)*2)))
		packBMPAVX2(chunk4, chunk5, order).StoreArray((*[16]uint16)(unsafe.Add(outputBase, uintptr(i+32)*2)))
		packBMPAVX2(chunk6, chunk7, order).StoreArray((*[16]uint16)(unsafe.Add(outputBase, uintptr(i+48)*2)))
	}
	for ; len(s)-i >= 4*encodeSIMDChunkSize; i += 4 * encodeSIMDChunkSize {
		chunk0 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
		chunk1 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+8)*4)))
		packBMPAVX2(chunk0, chunk1, order).StoreArray((*[16]uint16)(unsafe.Add(outputBase, uintptr(i)*2)))
	}
	for ; i < len(s); i++ {
		out[i] = uint16(s[i])
	}
	return out[:len(s)]
}
