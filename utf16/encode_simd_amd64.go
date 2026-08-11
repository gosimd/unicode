//go:build goexperiment.simd && amd64

package utf16

import (
	"math/bits"
	"simd/archsimd"
	stdutf16 "unicode/utf16"
	"unsafe"
)

const encodeSIMDChunkSize = 4

// encode uses AVX2 for eight-rune blocks that are all valid BMP code points
// outside the surrogate range. All other blocks use the scalar encoder, so
// the result and capacity match unicode/utf16.Encode for malformed runes too.
func encode(s []rune) []uint16 {
	if !archsimd.X86.AVX2() {
		return stdutf16.Encode(s)
	}

	capacity := encodedLengthSIMD(s)
	out := make([]uint16, capacity)
	return encodeSIMD(s, out, capacity)
}

// encodeSIMD encodes s into out. out must have room for the maximum encoded
// length, including an additional code unit for every rune at or above U+10000.
func encodeSIMD(s []rune, out []uint16, outputCapacity int) []uint16 {
	if len(out) < outputCapacity {
		panic("utf16: output buffer too small")
	}

	bmpBeforeSurrogates := archsimd.BroadcastUint32x4(surrogateHighStart)
	bmpAfterSurrogates := archsimd.BroadcastUint32x4(surrogateEnd)
	needsSurrogates := archsimd.BroadcastUint32x4(surrogateOffset)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	i, n := 0, 0
	for i < len(s) {
		if len(s)-i >= 2*encodeSIMDChunkSize {
			chunk0 := archsimd.LoadUint32x4Array((*[4]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
			chunk1 := archsimd.LoadUint32x4Array((*[4]uint32)(unsafe.Add(inputBase, uintptr(i+encodeSIMDChunkSize)*4)))
			if cleanBMPChunk(chunk0, bmpBeforeSurrogates, bmpAfterSurrogates, needsSurrogates) &&
				cleanBMPChunk(chunk1, bmpBeforeSurrogates, bmpAfterSurrogates, needsSurrogates) {
				// The clean-BMP predicate guarantees all values fit in uint16,
				// so saturating packing is identical to truncating narrowing.
				chunk0.BitsToInt32().SaturateToUint16Concat(chunk1.BitsToInt32()).Store(out[n:])
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

func encodedLengthSIMD(s []rune) int {
	n := len(s)
	threshold := archsimd.BroadcastInt32x4(surrogateOffset)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	i := 0
	for ; len(s)-i >= encodeSIMDChunkSize; i += encodeSIMDChunkSize {
		chunk := archsimd.LoadInt32x4Array((*[4]int32)(unsafe.Add(inputBase, uintptr(i)*4)))
		n += bits.OnesCount8(chunk.GreaterEqual(threshold).ToBits())
	}
	for ; i < len(s); i++ {
		if s[i] >= surrogateOffset {
			n++
		}
	}
	return n
}

func cleanBMPChunk(chunk, beforeSurrogates, afterSurrogates, needsSurrogates archsimd.Uint32x4) bool {
	clean := chunk.Less(beforeSurrogates).Or(
		chunk.GreaterEqual(afterSurrogates).And(chunk.Less(needsSurrogates)),
	)
	return clean.ToBits() == 0x0F
}
