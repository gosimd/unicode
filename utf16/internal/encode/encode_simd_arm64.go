//go:build goexperiment.simd && arm64

package encode

import (
	"simd/archsimd"
	"unsafe"
)

const encodeSIMDChunkSize = 4

// Encode uses NEON for eight-rune blocks that are all valid BMP code points
// outside the surrogate range. All other blocks use the scalar encoder, so
// the result and capacity match unicode/utf16.Encode for malformed runes too.
func Encode(s []rune) []uint16 {
	plan := planEncodeSIMD(s)
	out := make([]uint16, plan.capacity)
	return encodeSIMDWithPlan(s, out, plan)
}

func planEncodeSIMD(s []rune) encodingPlan {
	return encodingPlan{capacity: encodedLengthSIMD(s)}
}

func encodeSIMDWithPlan(s []rune, out []uint16, plan encodingPlan) []uint16 {
	return encodeSIMD(s, out, plan.capacity)
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
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0
	for i < len(s) {
		if len(s)-i >= 2*encodeSIMDChunkSize {
			chunk0 := archsimd.LoadUint32x4Array((*[4]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
			chunk1 := archsimd.LoadUint32x4Array((*[4]uint32)(unsafe.Add(inputBase, uintptr(i+encodeSIMDChunkSize)*4)))
			if cleanBMPChunk(chunk0, bmpBeforeSurrogates, bmpAfterSurrogates, needsSurrogates) &&
				cleanBMPChunk(chunk1, bmpBeforeSurrogates, bmpAfterSurrogates, needsSurrogates) {
				packed := chunk0.TruncToUint16().ReshapeToUint64s().
					InterleaveLo(chunk1.TruncToUint16().ReshapeToUint64s()).
					ReshapeToUint16s()
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

func encodedLengthSIMD(s []rune) int {
	n := len(s)
	threshold := archsimd.BroadcastInt32x4(surrogateOffset)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	i := 0
	for ; len(s)-i >= encodeSIMDChunkSize; i += encodeSIMDChunkSize {
		chunk := archsimd.LoadInt32x4Array((*[4]int32)(unsafe.Add(inputBase, uintptr(i)*4)))
		n += int(chunk.GreaterEqual(threshold).ToInt32x4().Neg().ReduceSum())
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
	return clean.ToInt32x4().ReduceMax() < 0
}
