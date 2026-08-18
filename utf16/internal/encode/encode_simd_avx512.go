//go:build goexperiment.simd && amd64

package encode

import (
	"math/bits"
	"simd/archsimd"
	"unsafe"
)

type encodeAVX512Mode uint8

const (
	encodeAVX512Scalar encodeAVX512Mode = iota
	encodeAVX512Width8
	encodeAVX512Width16
	encodeAVX512Width64
)

// encodeAVX512 encodes clean BMP runes sixteen at a time. TruncToUint16 emits
// AVX-512 narrowing and is therefore isolated from the AVX2 implementation.
func encodeAVX512(s []rune, out []uint16, outputCapacity int, mode encodeAVX512Mode) []uint16 {
	if len(out) < outputCapacity {
		panic("utf16: output buffer too small")
	}
	if mode == encodeAVX512Scalar {
		_, n := encodeScalar(s, out, 0, 0, len(s))
		return out[:n]
	}
	if mode == encodeAVX512Width8 {
		return encodeAVX2Short(s, out, outputCapacity)
	}

	bmpBeforeSurrogates := archsimd.BroadcastUint32x16(surrogateHighStart)
	bmpAfterSurrogates := archsimd.BroadcastUint32x16(surrogateEnd)
	needsSurrogates := archsimd.BroadcastUint32x16(surrogateOffset)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0
	for i < len(s) {
		if mode == encodeAVX512Width64 && len(s)-i >= 16*encodeSIMDChunkSize {
			// Four independent 16-rune iterations share one AVX-512 mask
			// extraction before their narrowing stores.
			chunk0 := archsimd.LoadUint32x16Array((*[16]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
			chunk1 := archsimd.LoadUint32x16Array((*[16]uint32)(unsafe.Add(inputBase, uintptr(i+16)*4)))
			chunk2 := archsimd.LoadUint32x16Array((*[16]uint32)(unsafe.Add(inputBase, uintptr(i+32)*4)))
			chunk3 := archsimd.LoadUint32x16Array((*[16]uint32)(unsafe.Add(inputBase, uintptr(i+48)*4)))
			clean := cleanBMPMaskAVX512(chunk0, bmpBeforeSurrogates, bmpAfterSurrogates, needsSurrogates).
				And(cleanBMPMaskAVX512(chunk1, bmpBeforeSurrogates, bmpAfterSurrogates, needsSurrogates)).
				And(cleanBMPMaskAVX512(chunk2, bmpBeforeSurrogates, bmpAfterSurrogates, needsSurrogates)).
				And(cleanBMPMaskAVX512(chunk3, bmpBeforeSurrogates, bmpAfterSurrogates, needsSurrogates))
			if clean.ToBits() == 0xFFFF {
				chunk0.TruncToUint16().StoreArray((*[16]uint16)(unsafe.Add(outputBase, uintptr(n)*2)))
				chunk1.TruncToUint16().StoreArray((*[16]uint16)(unsafe.Add(outputBase, uintptr(n+16)*2)))
				chunk2.TruncToUint16().StoreArray((*[16]uint16)(unsafe.Add(outputBase, uintptr(n+32)*2)))
				chunk3.TruncToUint16().StoreArray((*[16]uint16)(unsafe.Add(outputBase, uintptr(n+48)*2)))
				i += 16 * encodeSIMDChunkSize
				n += 16 * encodeSIMDChunkSize
				continue
			}
		}

		if len(s)-i >= 4*encodeSIMDChunkSize {
			chunk := archsimd.LoadUint32x16Array((*[16]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
			if cleanBMPMaskAVX512(chunk, bmpBeforeSurrogates, bmpAfterSurrogates, needsSurrogates).ToBits() == 0xFFFF {
				chunk.TruncToUint16().StoreArray((*[16]uint16)(unsafe.Add(outputBase, uintptr(n)*2)))
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

func encodedLengthAVX512(s []rune) int {
	capacity, _ := encodedLengthAVX512Profile(s)
	return capacity
}

// encodedLengthAVX512Profile computes the exact output capacity and chooses
// the clean-BMP window size for the encoding pass. The capacity pass already
// counts every non-BMP rune, so mode selection adds no second predicate pass
// and no per-input bitmap allocation.
func encodedLengthAVX512Profile(s []rune) (int, encodeAVX512Mode) {
	n := len(s)
	threshold := archsimd.BroadcastInt32x16(surrogateOffset)
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	i := 0
	for ; len(s)-i >= 4*encodeSIMDChunkSize; i += 4 * encodeSIMDChunkSize {
		chunk := archsimd.LoadInt32x16Array((*[16]int32)(unsafe.Add(inputBase, uintptr(i)*4)))
		n += bits.OnesCount16(chunk.GreaterEqual(threshold).ToBits())
	}
	for ; i < len(s); i++ {
		if s[i] >= surrogateOffset {
			n++
		}
	}

	// Pick by the non-BMP density already discovered by the capacity pass.
	// The conservative 1/128 and 1/32 cutoffs avoid long probes for ordinary
	// sparse text; denser input falls back to the short 8-rune path or scalar.
	extra := n - len(s)
	switch {
	case extra == 0:
		return n, encodeAVX512Width64
	case extra*128 <= len(s):
		return n, encodeAVX512Width64
	case extra*32 <= len(s):
		return n, encodeAVX512Width16
	case extra*8 <= len(s):
		return n, encodeAVX512Width8
	default:
		return n, encodeAVX512Scalar
	}
}

func cleanBMPMaskAVX512(chunk, beforeSurrogates, afterSurrogates, needsSurrogates archsimd.Uint32x16) archsimd.Mask32x16 {
	return chunk.Less(beforeSurrogates).Or(
		chunk.GreaterEqual(afterSurrogates).And(chunk.Less(needsSurrogates)),
	)
}
