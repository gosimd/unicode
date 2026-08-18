//go:build goexperiment.simd && amd64

package utf8

import (
	"simd/archsimd"
	stdutf8 "unicode/utf8"
	"unsafe"
)

const (
	decodeAVX2Window       = 64
	decodeAVX2Lookahead    = 16
	decodeAVX2ProcessBytes = 12

	decodeAVX2DenseTwoByteContinuations  = uint64(0xaaaaaaaaaaaaaaaa)
	decodeAVX2DenseFourByteContinuations = uint64(0xeeeeeeeeeeeeeeee)
	decodeAVX2DenseThreeByte48Mask       = uint64(0x0000db6db6db6db6)
	decodeAVX2Low48Mask                  = uint64(0x0000ffffffffffff)
)

type decodeAVX2ShuffleRow struct {
	indices    [16]uint8
	correction [4]uint32
}

var decodeAVX2Entries, decodeAVX2ShuffleRows = buildDecodeAVX2Tables()

var decodeAVX2Vectors = struct {
	low7, middle6, high6, first3   [4]uint32
	dense2Indices                  [16]uint8
	dense2Low, dense2High          [8]uint16
	dense2WideLead, dense2WideLast [16]uint16
	dense4Last, dense4Middle       [8]uint32
	dense4High, dense4First        [8]uint32
	dense3Lead, dense3Last         [16]uint8
}{
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
	dense2WideLead: [16]uint16{
		0x001f, 0x001f, 0x001f, 0x001f, 0x001f, 0x001f, 0x001f, 0x001f,
		0x001f, 0x001f, 0x001f, 0x001f, 0x001f, 0x001f, 0x001f, 0x001f,
	},
	dense2WideLast: [16]uint16{
		0x003f, 0x003f, 0x003f, 0x003f, 0x003f, 0x003f, 0x003f, 0x003f,
		0x003f, 0x003f, 0x003f, 0x003f, 0x003f, 0x003f, 0x003f, 0x003f,
	},
	dense4Last:   [8]uint32{0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f},
	dense4Middle: [8]uint32{0x0fc0, 0x0fc0, 0x0fc0, 0x0fc0, 0x0fc0, 0x0fc0, 0x0fc0, 0x0fc0},
	dense4High:   [8]uint32{0x3f000, 0x3f000, 0x3f000, 0x3f000, 0x3f000, 0x3f000, 0x3f000, 0x3f000},
	dense4First:  [8]uint32{0x1c0000, 0x1c0000, 0x1c0000, 0x1c0000, 0x1c0000, 0x1c0000, 0x1c0000, 0x1c0000},
	dense3Lead: [16]uint8{
		0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f,
		0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f,
	},
	dense3Last: [16]uint8{
		0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f,
		0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f,
	},
}

var decodeAVX2Dense3Shuffle = buildDecodeAVX2Dense3Shuffle()

// decodeValidAVX2 decodes already-validated UTF-8 into caller-owned output.
// It uses a 64-byte ASCII path and a table-driven 12-byte mixed-text path.
func decodeValidAVX2(input []byte, out []rune, outputRunes int) []rune {
	if len(out) < outputRunes {
		panic("utf8: decode output buffer too small")
	}

	inputBase := unsafe.Pointer(unsafe.SliceData(input))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0

	for len(input)-i >= decodeAVX2Window+decodeAVX2Lookahead {
		windowStart := i
		chunk0 := loadDecodeChunkAVX2(inputBase, windowStart)
		chunk1 := loadDecodeChunkAVX2(inputBase, windowStart+16)
		chunk2 := loadDecodeChunkAVX2(inputBase, windowStart+32)
		chunk3 := loadDecodeChunkAVX2(inputBase, windowStart+48)

		if allASCIIBlock(chunk0, chunk1, chunk2, chunk3) {
			storeASCIIBytes16AVX2(chunk0, unsafe.Add(outputBase, uintptr(n)*4))
			storeASCIIBytes16AVX2(chunk1, unsafe.Add(outputBase, uintptr(n+16)*4))
			storeASCIIBytes16AVX2(chunk2, unsafe.Add(outputBase, uintptr(n+32)*4))
			storeASCIIBytes16AVX2(chunk3, unsafe.Add(outputBase, uintptr(n+48)*4))
			i += decodeAVX2Window
			n += decodeAVX2Window
			continue
		}

		continuations := uint64(continuationMask(chunk0).ToBits()) |
			uint64(continuationMask(chunk1).ToBits())<<16 |
			uint64(continuationMask(chunk2).ToBits())<<32 |
			uint64(continuationMask(chunk3).ToBits())<<48
		switch continuations {
		case decodeAVX2DenseTwoByteContinuations:
			storeDenseTwoByteBlockAVX2(inputBase, i, unsafe.Add(outputBase, uintptr(n)*4))
			i += 64
			n += 32
			continue
		case decodeAVX2DenseFourByteContinuations:
			storeDenseFourByteBlockAVX2(inputBase, i, unsafe.Add(outputBase, uintptr(n)*4))
			i += 64
			n += 16
			continue
		}
		if continuations&decodeAVX2Low48Mask == decodeAVX2DenseThreeByte48Mask {
			storeDenseThreeByteBlockAVX2(inputBase, i, unsafe.Add(outputBase, uintptr(n)*4))
			i += 48
			n += 16
			continue
		}
		endMask := (^continuations) >> 1
		maxStart := windowStart + decodeAVX2Window - decodeAVX2ProcessBytes
		for i < maxStart {
			chunk := loadDecodeChunkAVX2(inputBase, i)
			consumed, produced := decodeMasked12AVX2(
				chunk,
				uint16(endMask)&0x0fff,
				unsafe.Add(outputBase, uintptr(n)*4),
			)
			i += consumed
			n += produced
			endMask >>= consumed
		}
	}

	for len(input)-i >= decodeAVX2Lookahead {
		chunk := loadDecodeChunkAVX2(inputBase, i)
		if chunk.BitsToInt8().Less(archsimd.Int8x16{}).ToBits() == 0 {
			storeASCIIBytes16AVX2(chunk, unsafe.Add(outputBase, uintptr(n)*4))
			i += 16
			n += 16
			continue
		}

		endMask := (^continuationMask(chunk).ToBits()) >> 1
		// The dense two-byte path writes eight lanes while the first twelve
		// bytes produce six. Avoid an exact-capacity overrun at the final edge.
		if endMask&0x0fff == 0x0aaa && n+8 > outputRunes {
			break
		}
		consumed, produced := decodeMasked12AVX2(
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

func loadDecodeChunkAVX2(base unsafe.Pointer, i int) archsimd.Uint8x16 {
	return archsimd.LoadUint8x16Array((*[16]uint8)(unsafe.Add(base, uintptr(i))))
}

func decodeMasked12AVX2(chunk archsimd.Uint8x16, endMask uint16, output unsafe.Pointer) (consumed, produced int) {
	switch endMask {
	case 0x0fff:
		storeASCIIBytes12AVX2(chunk, output)
		return 12, 12
	case 0x0aaa:
		storeDenseTwoByteRunesAVX2(chunk, output)
		return 12, 6
	}

	entry := decodeAVX2Entries[endMask]
	consumed = int(entry >> 8 & 0x0f)
	if consumed == 0 {
		panic("utf8: invalid AVX2 decode mask")
	}
	produced = 3 + int(entry>>12&1)
	row := &decodeAVX2ShuffleRows[uint8(entry)]
	packed := chunk.PermuteOrZero(
		archsimd.LoadUint8x16Array(&row.indices).BitsToInt8(),
	).ReshapeToUint32s()

	last := packed.ShiftAllRight(24).And(archsimd.LoadUint32x4Array(&decodeAVX2Vectors.low7))
	middle := packed.ShiftAllRight(10).And(archsimd.LoadUint32x4Array(&decodeAVX2Vectors.middle6))
	high := packed.ShiftAllLeft(4).And(archsimd.LoadUint32x4Array(&decodeAVX2Vectors.high6))
	first := packed.And(archsimd.LoadUint32x4Array(&decodeAVX2Vectors.first3)).ShiftAllLeft(18)
	decoded := last.Or(middle).Or(high).Or(first).
		Sub(archsimd.LoadUint32x4Array(&row.correction))
	decoded.BitsToInt32().StoreArray((*[4]int32)(output))
	return consumed, produced
}

func storeASCIIBytes16AVX2(chunk archsimd.Uint8x16, output unsafe.Pointer) {
	widened := chunk.ExtendToUint16()
	storeUint16x8AsRunesAVX2(widened.GetLo(), output)
	storeUint16x8AsRunesAVX2(widened.GetHi(), unsafe.Add(output, 8*4))
}

func storeASCIIBytes12AVX2(chunk archsimd.Uint8x16, output unsafe.Pointer) {
	widened := chunk.ExtendToUint16()
	storeUint16x8AsRunesAVX2(widened.GetLo(), output)
	widened.GetHi().ExtendLo4ToUint32().BitsToInt32().
		StoreArray((*[4]int32)(unsafe.Add(output, 8*4)))
}

func storeUint16x8AsRunesAVX2(chunk archsimd.Uint16x8, output unsafe.Pointer) {
	chunk.ExtendToUint32().BitsToInt32().StoreArray((*[8]int32)(output))
}

func storeDenseTwoByteRunesAVX2(chunk archsimd.Uint8x16, output unsafe.Pointer) {
	pairs := chunk.PermuteOrZero(
		archsimd.LoadUint8x16Array(&decodeAVX2Vectors.dense2Indices).BitsToInt8(),
	).ReshapeToUint16s()
	decoded := pairs.And(archsimd.LoadUint16x8Array(&decodeAVX2Vectors.dense2Low)).Or(
		pairs.ShiftAllRight(2).And(archsimd.LoadUint16x8Array(&decodeAVX2Vectors.dense2High)),
	)
	storeUint16x8AsRunesAVX2(decoded, output)
}

func storeDenseTwoByteBlockAVX2(inputBase unsafe.Pointer, i int, output unsafe.Pointer) {
	lead := archsimd.LoadUint16x16Array(&decodeAVX2Vectors.dense2WideLead)
	last := archsimd.LoadUint16x16Array(&decodeAVX2Vectors.dense2WideLast)
	for offset := 0; offset < 64; offset += 32 {
		words := archsimd.LoadUint8x32Array(
			(*[32]uint8)(unsafe.Add(inputBase, uintptr(i+offset))),
		).AsUint16x16()
		decoded := words.And(lead).ShiftAllLeft(6).
			Or(words.ShiftAllRight(8).And(last))
		storeUint16x8AsRunesAVX2(decoded.GetLo(), unsafe.Add(output, uintptr(offset/2)*4))
		storeUint16x8AsRunesAVX2(decoded.GetHi(), unsafe.Add(output, uintptr(offset/2+8)*4))
	}
}

func storeDenseFourByteBlockAVX2(inputBase unsafe.Pointer, i int, output unsafe.Pointer) {
	lastMask := archsimd.LoadUint32x8Array(&decodeAVX2Vectors.dense4Last)
	middleMask := archsimd.LoadUint32x8Array(&decodeAVX2Vectors.dense4Middle)
	highMask := archsimd.LoadUint32x8Array(&decodeAVX2Vectors.dense4High)
	firstMask := archsimd.LoadUint32x8Array(&decodeAVX2Vectors.dense4First)
	for offset := 0; offset < 64; offset += 32 {
		words := archsimd.LoadUint8x32Array(
			(*[32]uint8)(unsafe.Add(inputBase, uintptr(i+offset))),
		).AsUint32x8()
		decoded := words.ShiftAllRight(24).And(lastMask).
			Or(words.ShiftAllRight(10).And(middleMask)).
			Or(words.ShiftAllLeft(4).And(highMask)).
			Or(words.ShiftAllLeft(18).And(firstMask))
		decoded.BitsToInt32().StoreArray(
			(*[8]int32)(unsafe.Add(output, uintptr(offset/4)*4)),
		)
	}
}

func storeDenseThreeByteBlockAVX2(inputBase unsafe.Pointer, i int, output unsafe.Pointer) {
	low4 := archsimd.LoadUint8x16Array(&decodeAVX2Vectors.dense3Lead)
	low6 := archsimd.LoadUint8x16Array(&decodeAVX2Vectors.dense3Last)
	for offset := 0; offset < 48; offset += 24 {
		chunk0 := loadDecodeChunkAVX2(inputBase, i+offset)
		chunk1 := loadDecodeChunkAVX2(inputBase, i+offset+16)
		lead := chunk0.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX2Dense3Shuffle[0][0]).BitsToInt8(),
		).Or(chunk1.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX2Dense3Shuffle[0][1]).BitsToInt8(),
		))
		middle := chunk0.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX2Dense3Shuffle[1][0]).BitsToInt8(),
		).Or(chunk1.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX2Dense3Shuffle[1][1]).BitsToInt8(),
		))
		last := chunk0.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX2Dense3Shuffle[2][0]).BitsToInt8(),
		).Or(chunk1.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX2Dense3Shuffle[2][1]).BitsToInt8(),
		))
		decoded := lead.And(low4).ExtendLo8ToUint32().ShiftAllLeft(12).
			Or(middle.And(low6).ExtendLo8ToUint32().ShiftAllLeft(6)).
			Or(last.And(low6).ExtendLo8ToUint32())
		decoded.BitsToInt32().StoreArray(
			(*[8]int32)(unsafe.Add(output, uintptr(offset/3)*4)),
		)
	}
}

func buildDecodeAVX2Dense3Shuffle() (tables [3][2][16]uint8) {
	for byteInRune := range tables {
		for source := range tables[byteInRune] {
			for lane := range tables[byteInRune][source] {
				tables[byteInRune][source][lane] = 0xff
			}
		}
	}
	for lane := range 8 {
		for byteInRune := range 3 {
			position := 3*lane + byteInRune
			source := position / 16
			tables[byteInRune][source][lane] = uint8(position % 16)
		}
	}
	return tables
}

func buildDecodeAVX2Tables() ([4096]uint16, [256]decodeAVX2ShuffleRow) {
	var entries [4096]uint16
	var rows [256]decodeAVX2ShuffleRow

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
		for bit := 0; bit < decodeAVX2ProcessBytes && endCount < len(ends); bit++ {
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
