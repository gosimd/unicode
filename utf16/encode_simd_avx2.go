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
)

// encodeAVX2 encodes clean BMP runes sixteen at a time. Its packing uses AVX2
// VPACKUSDW, so it is safe on AVX2-only CPUs.
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
	return encodeAVX2Wide(s, out, outputCapacity)
}

func encodeAVX2Wide(s []rune, out []uint16, outputCapacity int) []uint16 {

	bmpBeforeSurrogates := archsimd.BroadcastUint32x8(surrogateHighStart)
	bmpAfterSurrogates := archsimd.BroadcastUint32x8(surrogateEnd)
	needsSurrogates := archsimd.BroadcastUint32x8(surrogateOffset)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0
	for i < len(s) {
		if len(s)-i >= 4*encodeSIMDChunkSize {
			chunk0 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
			chunk1 := archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(inputBase, uintptr(i+8)*4)))
			clean := cleanBMPMaskAVX2Wide(chunk0, bmpBeforeSurrogates, bmpAfterSurrogates, needsSurrogates).
				And(cleanBMPMaskAVX2Wide(chunk1, bmpBeforeSurrogates, bmpAfterSurrogates, needsSurrogates))
			if clean.ToBits() == 0xFF {
				// VPACKUSDW operates separately on 128-bit lanes. Rearranging
				// sources first makes its grouped result sequential in memory.
				lo := chunk0.ConcatPermute128Scalars(0, 2, chunk1)
				hi := chunk0.ConcatPermute128Scalars(1, 3, chunk1)
				lo.BitsToInt32().SaturateToUint16ConcatGrouped(hi.BitsToInt32()).
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

func cleanBMPMaskAVX2Wide(chunk, beforeSurrogates, afterSurrogates, needsSurrogates archsimd.Uint32x8) archsimd.Mask32x8 {
	return chunk.Less(beforeSurrogates).Or(
		chunk.GreaterEqual(afterSurrogates).And(chunk.Less(needsSurrogates)),
	)
}

// encodeAVX2Short is the original 8-rune AVX2 loop. AVX-512 uses it when the
// length pass observes clean 8-rune windows but not clean 16-rune windows.
func encodeAVX2Short(s []rune, out []uint16, outputCapacity int) []uint16 {
	if len(out) < outputCapacity {
		panic("utf16: output buffer too small")
	}

	bmpBeforeSurrogates := archsimd.BroadcastUint32x4(surrogateHighStart)
	bmpAfterSurrogates := archsimd.BroadcastUint32x4(surrogateEnd)
	needsSurrogates := archsimd.BroadcastUint32x4(surrogateOffset)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0
	for i < len(s) {
		if len(s)-i >= 2*encodeSIMDChunkSize {
			chunk0 := archsimd.LoadUint32x4Array((*[4]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
			chunk1 := archsimd.LoadUint32x4Array((*[4]uint32)(unsafe.Add(inputBase, uintptr(i+encodeSIMDChunkSize)*4)))
			clean := cleanBMPMaskAVX2(chunk0, bmpBeforeSurrogates, bmpAfterSurrogates, needsSurrogates).
				And(cleanBMPMaskAVX2(chunk1, bmpBeforeSurrogates, bmpAfterSurrogates, needsSurrogates))
			if clean.ToBits() == 0x0F {
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

// encodedLengthAVX2Profile chooses a width from the non-BMP density already
// counted for capacity. It avoids wide probes on sparse and scalar-heavy text
// without an additional input pass.
func encodedLengthAVX2Profile(s []rune) (int, encodeAVX2Mode) {
	n := len(s)
	threshold := archsimd.BroadcastInt32x8(surrogateOffset)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	i := 0
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

func cleanBMPMaskAVX2(chunk, beforeSurrogates, afterSurrogates, needsSurrogates archsimd.Uint32x4) archsimd.Mask32x4 {
	return chunk.Less(beforeSurrogates).Or(
		chunk.GreaterEqual(afterSurrogates).And(chunk.Less(needsSurrogates)),
	)
}
