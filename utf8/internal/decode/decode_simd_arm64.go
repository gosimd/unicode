//go:build goexperiment.simd && arm64

package decode

import (
	"github.com/gosimd/unicode/utf8/internal/scan"
	"simd/archsimd"
	stdutf8 "unicode/utf8"
	"unsafe"
)

const (
	decodeSIMDWindow       = 64
	decodeSIMDLookahead    = 16
	decodeSIMDProcessBytes = 12
)

type decodeShuffleRow struct {
	indices    [16]uint8
	correction [4]uint32
}

var decodeEntries, decodeShuffleRows = buildDecodeTables()

var decodeVectors = struct {
	bitWeights                   [16]uint8
	low7, middle6, high6, first3 [4]uint32
	dense2Indices                [16]uint8
	dense2Low, dense2High        [8]uint16
}{
	bitWeights: [16]uint8{
		1, 2, 4, 8, 16, 32, 64, 128,
		1, 2, 4, 8, 16, 32, 64, 128,
	},
	low7:    [4]uint32{0x7f, 0x7f, 0x7f, 0x7f},
	middle6: [4]uint32{0x0fc0, 0x0fc0, 0x0fc0, 0x0fc0},
	high6:   [4]uint32{0x3f000, 0x3f000, 0x3f000, 0x3f000},
	first3:  [4]uint32{0x07, 0x07, 0x07, 0x07},
	dense2Indices: [16]uint8{
		1, 0, 3, 2, 5, 4, 7, 6,
		9, 8, 11, 10, 0xff, 0xff, 0xff, 0xff,
	},
	dense2Low:  [8]uint16{0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f},
	dense2High: [8]uint16{0x7c0, 0x7c0, 0x7c0, 0x7c0, 0x7c0, 0x7c0, 0x7c0, 0x7c0},
}

// Decode returns the runes in s. Invalid UTF-8 encodings are replaced by
// RuneError, as in the language conversion []rune(s).
func Decode(s string) []rune {
	if len(s) == 0 {
		return []rune(s)
	}

	input := unsafe.Slice(unsafe.StringData(s), len(s))
	count, valid := scan.CountValid(input)
	if !valid {
		return []rune(s)
	}

	out := make([]rune, count)
	decodeValidSIMD(input, out, count)
	return out
}

// decodeValidSIMD decodes already-validated UTF-8 into caller-owned output.
// out must contain exactly outputRunes writable lanes.
func decodeValidSIMD(input []byte, out []rune, outputRunes int) []rune {
	if len(out) < outputRunes {
		panic("utf8: decode output buffer too small")
	}

	inputBase := unsafe.Pointer(unsafe.SliceData(input))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0

	for len(input)-i >= decodeSIMDWindow+decodeSIMDLookahead {
		windowStart := i
		chunk0 := loadDecodeChunk(inputBase, windowStart)
		chunk1 := loadDecodeChunk(inputBase, windowStart+16)
		chunk2 := loadDecodeChunk(inputBase, windowStart+32)
		chunk3 := loadDecodeChunk(inputBase, windowStart+48)

		if allASCIIBlock(chunk0, chunk1, chunk2, chunk3) {
			storeASCIIBytes16(chunk0, unsafe.Add(outputBase, uintptr(n)*4))
			storeASCIIBytes16(chunk1, unsafe.Add(outputBase, uintptr(n+16)*4))
			storeASCIIBytes16(chunk2, unsafe.Add(outputBase, uintptr(n+32)*4))
			storeASCIIBytes16(chunk3, unsafe.Add(outputBase, uintptr(n+48)*4))
			i += decodeSIMDWindow
			n += decodeSIMDWindow
			continue
		}

		continuations := uint64(continuationBits(chunk0)) |
			uint64(continuationBits(chunk1))<<16 |
			uint64(continuationBits(chunk2))<<32 |
			uint64(continuationBits(chunk3))<<48
		endMask := (^continuations) >> 1
		maxStart := windowStart + decodeSIMDWindow - decodeSIMDProcessBytes
		for i < maxStart {
			chunk := loadDecodeChunk(inputBase, i)
			consumed, produced := decodeMasked12(
				chunk,
				uint16(endMask)&0x0fff,
				unsafe.Add(outputBase, uintptr(n)*4),
			)
			i += consumed
			n += produced
			endMask >>= consumed
		}
	}

	for len(input)-i >= decodeSIMDLookahead {
		chunk := loadDecodeChunk(inputBase, i)
		if chunk.BitsToInt8().ReduceMin() >= 0 {
			storeASCIIBytes16(chunk, unsafe.Add(outputBase, uintptr(n)*4))
			i += 16
			n += 16
			continue
		}

		endMask := (^continuationBits(chunk)) >> 1
		// The dense two-byte path widens and stores eight lanes although the
		// first twelve bytes produce six runes. A final four-byte rune can leave
		// only seven output lanes; finish that rare boundary scalar instead of
		// over-allocating every public result.
		if endMask&0x0fff == 0x0aaa && n+8 > outputRunes {
			break
		}
		consumed, produced := decodeMasked12(
			chunk,
			endMask&0x0fff,
			unsafe.Add(outputBase, uintptr(n)*4),
		)
		i += consumed
		n += produced
	}

	for i < len(input) {
		r, size := stdutf8.DecodeRune(input[i:])
		out[n] = r
		i += size
		n++
	}
	return out[:n]
}

func loadDecodeChunk(base unsafe.Pointer, i int) archsimd.Uint8x16 {
	return archsimd.LoadUint8x16Array((*[16]uint8)(unsafe.Add(base, uintptr(i))))
}

// continuationBits returns one scalar bit for every byte in 0x80..0xbf.
// archsimd has no ARM64 Mask8x16.ToBits, so the two halves are weighted and
// horizontally summed independently.
func continuationBits(chunk archsimd.Uint8x16) uint16 {
	ones := maskBits(chunk.BitsToInt8().Less(archsimd.BroadcastInt8x16(-64))).ShiftAllRight(7)
	weighted := ones.Mul(archsimd.LoadUint8x16Array(&decodeVectors.bitWeights))
	lo := weighted.ExtendLo8ToUint16().ReduceSum()
	hi := weighted.HiToLo().ExtendLo8ToUint16().ReduceSum()
	return uint16(lo) | uint16(hi)<<8
}

func decodeMasked12(chunk archsimd.Uint8x16, endMask uint16, output unsafe.Pointer) (consumed, produced int) {
	switch endMask {
	case 0x0fff:
		storeASCIIBytes12(chunk, output)
		return 12, 12
	case 0x0aaa:
		storeDenseTwoByteRunes(chunk, output)
		return 12, 6
	}

	entry := decodeEntries[endMask]
	consumed = int(entry >> 8 & 0x0f)
	if consumed == 0 {
		panic("utf8: invalid SIMD decode mask")
	}
	produced = 3 + int(entry>>12&1)
	row := &decodeShuffleRows[uint8(entry)]
	packed := chunk.LookupOrZero(archsimd.LoadUint8x16Array(&row.indices)).ReshapeToUint32s()

	last := packed.ShiftAllRight(24).And(archsimd.LoadUint32x4Array(&decodeVectors.low7))
	middle := packed.ShiftAllRight(10).And(archsimd.LoadUint32x4Array(&decodeVectors.middle6))
	high := packed.ShiftAllLeft(4).And(archsimd.LoadUint32x4Array(&decodeVectors.high6))
	first := packed.And(archsimd.LoadUint32x4Array(&decodeVectors.first3)).ShiftAllLeft(18)
	decoded := last.Or(middle).Or(high).Or(first).
		Sub(archsimd.LoadUint32x4Array(&row.correction))
	decoded.BitsToInt32().StoreArray((*[4]int32)(output))
	return consumed, produced
}

func storeASCIIBytes16(chunk archsimd.Uint8x16, output unsafe.Pointer) {
	lo := chunk.ExtendLo8ToUint16()
	hi := chunk.HiToLo().ExtendLo8ToUint16()
	storeUint16x8AsRunes(lo, output)
	storeUint16x8AsRunes(hi, unsafe.Add(output, 8*4))
}

func storeASCIIBytes12(chunk archsimd.Uint8x16, output unsafe.Pointer) {
	storeUint16x8AsRunes(chunk.ExtendLo8ToUint16(), output)
	chunk.HiToLo().ExtendLo8ToUint16().ExtendLo4ToUint32().BitsToInt32().
		StoreArray((*[4]int32)(unsafe.Add(output, 8*4)))
}

func storeUint16x8AsRunes(chunk archsimd.Uint16x8, output unsafe.Pointer) {
	chunk.ExtendLo4ToUint32().BitsToInt32().StoreArray((*[4]int32)(output))
	chunk.HiToLo().ExtendLo4ToUint32().BitsToInt32().
		StoreArray((*[4]int32)(unsafe.Add(output, 4*4)))
}

func storeDenseTwoByteRunes(chunk archsimd.Uint8x16, output unsafe.Pointer) {
	pairs := chunk.LookupOrZero(archsimd.LoadUint8x16Array(&decodeVectors.dense2Indices)).
		ReshapeToUint16s()
	decoded := pairs.And(archsimd.LoadUint16x8Array(&decodeVectors.dense2Low)).Or(
		pairs.ShiftAllRight(2).And(archsimd.LoadUint16x8Array(&decodeVectors.dense2High)),
	)
	storeUint16x8AsRunes(decoded, output)
}

func buildDecodeTables() ([4096]uint16, [256]decodeShuffleRow) {
	var entries [4096]uint16
	var rows [256]decodeShuffleRow

	for selector := range rows {
		row := &rows[selector]
		for i := range row.indices {
			row.indices[i] = 0xff
		}
		source := 0
		for lane := 0; lane < 4; lane++ {
			length := (selector>>(2*lane))&3 + 1
			destination := lane*4 + 4 - length
			for j := 0; j < length; j++ {
				row.indices[destination+j] = uint8(source + j)
			}
			if length == 3 {
				row.correction[lane] = 0x20000
			}
			source += length
		}
	}

	for mask := range entries {
		var ends [4]int
		endCount := 0
		for bit := 0; bit < decodeSIMDProcessBytes && endCount < len(ends); bit++ {
			if mask&(1<<bit) != 0 {
				ends[endCount] = bit
				endCount++
			}
		}
		if endCount < 3 {
			continue
		}

		produced := 4
		if endCount == 3 {
			produced = 3
		}
		selector := 0
		start := 0
		valid := true
		for lane := 0; lane < produced; lane++ {
			length := ends[lane] - start + 1
			if length < 1 || length > 4 {
				valid = false
				break
			}
			selector |= (length - 1) << (2 * lane)
			start = ends[lane] + 1
		}
		if !valid {
			continue
		}
		consumed := ends[produced-1] + 1
		entries[mask] = uint16(selector) |
			uint16(consumed)<<8 |
			uint16(produced-3)<<12
	}
	return entries, rows
}
