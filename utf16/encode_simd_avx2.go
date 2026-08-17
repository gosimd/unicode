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
	encodeAVX2AllNonBMP
	encodeAVX2MixedValid
)

var encodeAVX2PackOrder = [8]uint32{0, 1, 4, 5, 2, 3, 6, 7}

var encodeAVX2MixedShuffle = [16][16]int8{
	{0, 1, 4, 5, 8, 9, 12, 13, -1, -1, -1, -1, -1, -1, -1, -1},
	{0, 1, 2, 3, 4, 5, 8, 9, 12, 13, -1, -1, -1, -1, -1, -1},
	{0, 1, 4, 5, 6, 7, 8, 9, 12, 13, -1, -1, -1, -1, -1, -1},
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 13, -1, -1, -1, -1},
	{0, 1, 4, 5, 8, 9, 10, 11, 12, 13, -1, -1, -1, -1, -1, -1},
	{0, 1, 2, 3, 4, 5, 8, 9, 10, 11, 12, 13, -1, -1, -1, -1},
	{0, 1, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, -1, -1, -1, -1},
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, -1, -1},
	{0, 1, 4, 5, 8, 9, 12, 13, 14, 15, -1, -1, -1, -1, -1, -1},
	{0, 1, 2, 3, 4, 5, 8, 9, 12, 13, 14, 15, -1, -1, -1, -1},
	{0, 1, 4, 5, 6, 7, 8, 9, 12, 13, 14, 15, -1, -1, -1, -1},
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 13, 14, 15, -1, -1},
	{0, 1, 4, 5, 8, 9, 10, 11, 12, 13, 14, 15, -1, -1, -1, -1},
	{0, 1, 2, 3, 4, 5, 8, 9, 10, 11, 12, 13, 14, 15, -1, -1},
	{0, 1, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, -1, -1},
	{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15},
}

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
	if mode == encodeAVX2AllNonBMP {
		return encodeAVX2AllNonBMPOnly(s, out, outputCapacity)
	}
	if mode == encodeAVX2MixedValid {
		return encodeAVX2MixedValidOnly(s, out, outputCapacity)
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
	maximumRune := archsimd.BroadcastUint32x8(0x10FFFF)
	surrogateBits16 := archsimd.BroadcastUint16x16(surrogateMask)
	surrogateMarker16 := archsimd.BroadcastUint16x16(surrogateHighStart)
	zeroBytes := archsimd.BroadcastUint8x32(0)
	nonBMPCounts := archsimd.BroadcastUint64x4(0)
	allValid := true
	for ; len(s)-i >= 8*encodeSIMDChunkSize; i += 8 * encodeSIMDChunkSize {
		chunk0 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
		chunk1 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+8)*4)))
		chunk2 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+16)*4)))
		chunk3 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+24)*4)))
		mask0 := chunk0.BitsToInt32().GreaterEqual(threshold)
		mask1 := chunk1.BitsToInt32().GreaterEqual(threshold)
		mask2 := chunk2.BitsToInt32().GreaterEqual(threshold)
		mask3 := chunk3.BitsToInt32().GreaterEqual(threshold)
		laneCounts := mask0.ToInt32x8().Add(mask1.ToInt32x8()).
			Add(mask2.ToInt32x8()).Add(mask3.ToInt32x8()).Neg()
		nonBMPCounts = nonBMPCounts.Add(laneCounts.AsUint8x32().SumOf8AbsDiff(zeroBytes))
		if allValid {
			maximum := chunk0.Max(chunk1).Max(chunk2.Max(chunk3))
			packed01 := chunk0.BitsToInt32().SaturateToUint16ConcatGrouped(chunk1.BitsToInt32())
			packed23 := chunk2.BitsToInt32().SaturateToUint16ConcatGrouped(chunk3.BitsToInt32())
			surrogates := packed01.And(surrogateBits16).Equal(surrogateMarker16).ToInt16x16().AsUint16x16().
				Or(packed23.And(surrogateBits16).Equal(surrogateMarker16).ToInt16x16().AsUint16x16())
			tooHigh := maximum.Greater(maximumRune).ToInt32x8().AsUint16x16()
			allValid = tooHigh.Or(surrogates).IsZero()
		}
	}
	var counts [4]uint64
	nonBMPCounts.StoreArray(&counts)
	n += int(counts[0] + counts[1] + counts[2] + counts[3])
	surrogateBits := archsimd.BroadcastUint32x8(surrogateMask)
	surrogateMarker := archsimd.BroadcastUint32x8(surrogateHighStart)
	for ; len(s)-i >= 2*encodeSIMDChunkSize; i += 2 * encodeSIMDChunkSize {
		chunk := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
		n += bits.OnesCount8(chunk.BitsToInt32().GreaterEqual(threshold).ToBits())
		if allValid {
			allValid = chunk.Greater(maximumRune).
				Or(chunk.And(surrogateBits).Equal(surrogateMarker)).ToBits() == 0
		}
	}
	for ; i < len(s); i++ {
		r := s[i]
		if r >= surrogateOffset {
			n++
		}
		if uint32(r) > 0x10FFFF || uint32(r)-surrogateHighStart < surrogateEnd-surrogateHighStart {
			allValid = false
		}
	}

	extra := n - len(s)
	switch {
	case allValid && extra == len(s):
		return n, encodeAVX2AllNonBMP
	case allValid && extra*32 > len(s):
		return n, encodeAVX2MixedValid
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

func encodeNonBMPPairsAVX2(chunk, base, lowBits, shift10, shift16, pairBase archsimd.Uint32x8) archsimd.Uint32x8 {
	delta := chunk.Sub(base)
	return delta.ShiftRight(shift10).
		Add(delta.ShiftLeft(shift16).And(lowBits)).
		Add(pairBase)
}

//go:noinline
func encodeAVX2AllNonBMPOnly(s []rune, out []uint16, outputCapacity int) []uint16 {
	if len(out) < outputCapacity {
		panic("utf16: output buffer too small")
	}

	base := archsimd.BroadcastUint32x8(surrogateOffset)
	lowBits := archsimd.BroadcastUint32x8(0x03FF0000)
	shift10 := archsimd.BroadcastUint32x8(10)
	shift16 := archsimd.BroadcastUint32x8(16)
	pairBase := archsimd.BroadcastUint32x8(0xDC00D800)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i := 0
	for ; len(s)-i >= 8*encodeSIMDChunkSize; i += 8 * encodeSIMDChunkSize {
		chunk0 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
		chunk1 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+8)*4)))
		chunk2 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+16)*4)))
		chunk3 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+24)*4)))
		encodeNonBMPPairsAVX2(chunk0, base, lowBits, shift10, shift16, pairBase).
			StoreArray((*[8]uint32)(unsafe.Add(outputBase, uintptr(i)*4)))
		encodeNonBMPPairsAVX2(chunk1, base, lowBits, shift10, shift16, pairBase).
			StoreArray((*[8]uint32)(unsafe.Add(outputBase, uintptr(i+8)*4)))
		encodeNonBMPPairsAVX2(chunk2, base, lowBits, shift10, shift16, pairBase).
			StoreArray((*[8]uint32)(unsafe.Add(outputBase, uintptr(i+16)*4)))
		encodeNonBMPPairsAVX2(chunk3, base, lowBits, shift10, shift16, pairBase).
			StoreArray((*[8]uint32)(unsafe.Add(outputBase, uintptr(i+24)*4)))
	}
	for ; len(s)-i >= 2*encodeSIMDChunkSize; i += 2 * encodeSIMDChunkSize {
		chunk := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
		encodeNonBMPPairsAVX2(chunk, base, lowBits, shift10, shift16, pairBase).
			StoreArray((*[8]uint32)(unsafe.Add(outputBase, uintptr(i)*4)))
	}
	for ; i < len(s); i++ {
		r := s[i] - surrogateOffset
		out[2*i] = uint16(surrogateHighStart + (r >> 10))
		out[2*i+1] = uint16(surrogateLowStart + r&0x3FF)
	}
	return out[:outputCapacity]
}

func encodeNonBMPPairsAVX2Short(chunk, base, lowBits, shift10, shift16, pairBase archsimd.Uint32x4) archsimd.Uint32x4 {
	delta := chunk.Sub(base)
	return delta.ShiftRight(shift10).
		Add(delta.ShiftLeft(shift16).And(lowBits)).
		Add(pairBase)
}

//go:noinline
func encodeAVX2MixedValidOnly(s []rune, out []uint16, outputCapacity int) []uint16 {
	if len(out) < outputCapacity {
		panic("utf16: output buffer too small")
	}

	threshold := archsimd.BroadcastInt32x4(surrogateOffset)
	base := archsimd.BroadcastUint32x4(surrogateOffset)
	lowBits := archsimd.BroadcastUint32x4(0x03FF0000)
	shift10 := archsimd.BroadcastUint32x4(10)
	shift16 := archsimd.BroadcastUint32x4(16)
	pairBase := archsimd.BroadcastUint32x4(0xDC00D800)
	zero := archsimd.BroadcastInt32x4(0)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0
	// Every SIMD store writes eight code units. Keeping four input runes for
	// the tail makes the overlap fit in the exact-capacity output buffer.
	for ; len(s)-i >= 2*encodeSIMDChunkSize; i += encodeSIMDChunkSize {
		chunk := archsimd.LoadUint32x4Array((*[4]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
		nonBMP := chunk.BitsToInt32().GreaterEqual(threshold)
		mask := nonBMP.ToBits()
		if mask == 0 {
			chunk.BitsToInt32().SaturateToUint16Concat(zero).
				StoreArray((*[8]uint16)(unsafe.Add(outputBase, uintptr(n)*2)))
			n += encodeSIMDChunkSize
			continue
		}

		pairs := encodeNonBMPPairsAVX2Short(chunk, base, lowBits, shift10, shift16, pairBase)
		if mask != 0x0F {
			pairs = pairs.IfElse(nonBMP, chunk)
			shuffle := archsimd.LoadInt8x16Array(&encodeAVX2MixedShuffle[mask])
			pairs.ReshapeToUint8s().PermuteOrZero(shuffle).
				StoreArray((*[16]uint8)(unsafe.Add(outputBase, uintptr(n)*2)))
		} else {
			pairs.StoreArray((*[4]uint32)(unsafe.Add(outputBase, uintptr(n)*2)))
		}
		n += encodeSIMDChunkSize + bits.OnesCount8(mask)
	}
	_, n = encodeScalar(s, out, i, n, len(s))
	return out[:n]
}
