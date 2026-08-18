//go:build goexperiment.simd && amd64

package utf8

import (
	"simd/archsimd"
	stdutf8 "unicode/utf8"
	"unsafe"
)

const encodeAVX2Runes = 8

type encodeAVX2Shuffle struct {
	size    uint8
	indices [16]uint8
}

var (
	encodeAVX2Shuffles   = buildEncodeAVX2Shuffles()
	encodeAVX2LengthBits = buildEncodeAVX2LengthBits()
	encodeAVX2ASCII      = [16]int8{0, 2, 4, 6, 8, 10, 12, 14, -1, -1, -1, -1, -1, -1, -1, -1}
	encodeAVX2Vectors    = struct {
		threshold2, threshold3, threshold4 [8]uint32
		low6, continuation, low7           [8]uint32
		lead2, lead3, lead4                [8]uint32
	}{
		threshold2:   encodeAVX2Splat(0x80),
		threshold3:   encodeAVX2Splat(0x800),
		threshold4:   encodeAVX2Splat(0x10000),
		low6:         encodeAVX2Splat(0x3f),
		continuation: encodeAVX2Splat(0x80),
		low7:         encodeAVX2Splat(0x7f),
		lead2:        encodeAVX2Splat(0xc0),
		lead3:        encodeAVX2Splat(0xe0),
		lead4:        encodeAVX2Splat(0xf0),
	}
)

func planEncodeAVX2(s []rune) encodePlan {
	if allASCIIEncodeAVX2(s) {
		return encodePlan{size: len(s), allASCII: true, allValid: true}
	}

	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	maximumRune := archsimd.BroadcastUint32x8(stdutf8.MaxRune)
	surrogateBits := archsimd.BroadcastUint32x8(0xfffff800)
	surrogateMarker := archsimd.BroadcastUint32x8(0xd800)
	replacement := archsimd.BroadcastUint32x8(stdutf8.RuneError)
	threshold2 := archsimd.BroadcastInt32x8(0x80)
	threshold3 := archsimd.BroadcastInt32x8(0x800)
	threshold4 := archsimd.BroadcastInt32x8(0x10000)
	zeroBytes := archsimd.BroadcastUint8x32(0)
	var extraCounts archsimd.Uint64x4
	allValid := true
	i := 0
	for ; len(s)-i >= 4*encodeAVX2Runes; i += 4 * encodeAVX2Runes {
		chunk0 := loadEncodeAVX2Chunk(inputBase, i)
		invalid0 := chunk0.Greater(maximumRune).Or(chunk0.And(surrogateBits).Equal(surrogateMarker))
		invalidBits := invalid0.ToInt32x8()
		chunk0 = replacement.IfElse(invalid0, chunk0)
		laneCounts := encodeAVX2ExtraCounts(chunk0, threshold2, threshold3, threshold4)

		chunk1 := loadEncodeAVX2Chunk(inputBase, i+encodeAVX2Runes)
		invalid1 := chunk1.Greater(maximumRune).Or(chunk1.And(surrogateBits).Equal(surrogateMarker))
		invalidBits = invalidBits.Or(invalid1.ToInt32x8())
		chunk1 = replacement.IfElse(invalid1, chunk1)
		laneCounts = laneCounts.Add(encodeAVX2ExtraCounts(chunk1, threshold2, threshold3, threshold4))

		chunk2 := loadEncodeAVX2Chunk(inputBase, i+2*encodeAVX2Runes)
		invalid2 := chunk2.Greater(maximumRune).Or(chunk2.And(surrogateBits).Equal(surrogateMarker))
		invalidBits = invalidBits.Or(invalid2.ToInt32x8())
		chunk2 = replacement.IfElse(invalid2, chunk2)
		laneCounts = laneCounts.Add(encodeAVX2ExtraCounts(chunk2, threshold2, threshold3, threshold4))

		chunk3 := loadEncodeAVX2Chunk(inputBase, i+3*encodeAVX2Runes)
		invalid3 := chunk3.Greater(maximumRune).Or(chunk3.And(surrogateBits).Equal(surrogateMarker))
		invalidBits = invalidBits.Or(invalid3.ToInt32x8())
		chunk3 = replacement.IfElse(invalid3, chunk3)
		laneCounts = laneCounts.Add(encodeAVX2ExtraCounts(chunk3, threshold2, threshold3, threshold4))

		if allValid {
			allValid = invalidBits.IsZero()
		}
		laneCounts = laneCounts.Neg()
		extraCounts = extraCounts.Add(laneCounts.AsUint8x32().SumOf8AbsDiff(zeroBytes))
	}

	var counts [4]uint64
	extraCounts.StoreArray(&counts)
	size := len(s) + int(counts[0]+counts[1]+counts[2]+counts[3])
	for ; i < len(s); i++ {
		r := s[i]
		if r < 0 || r > stdutf8.MaxRune || r >= 0xd800 && r <= 0xdfff {
			r = stdutf8.RuneError
			allValid = false
		}
		switch {
		case r >= 0x10000:
			size += 3
		case r >= 0x800:
			size += 2
		case r >= 0x80:
			size++
		}
	}
	return encodePlan{size: size, allASCII: false, allValid: allValid}
}

func allASCIIEncodeAVX2(s []rune) bool {
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	nonASCIIBits := archsimd.BroadcastUint32x8(0xffffff80)
	i := 0
	for ; len(s)-i >= 4*encodeAVX2Runes; i += 4 * encodeAVX2Runes {
		chunk0 := loadEncodeAVX2Chunk(inputBase, i)
		chunk1 := loadEncodeAVX2Chunk(inputBase, i+encodeAVX2Runes)
		chunk2 := loadEncodeAVX2Chunk(inputBase, i+2*encodeAVX2Runes)
		chunk3 := loadEncodeAVX2Chunk(inputBase, i+3*encodeAVX2Runes)
		if !chunk0.Or(chunk1).Or(chunk2.Or(chunk3)).And(nonASCIIBits).IsZero() {
			return false
		}
	}
	for ; len(s)-i >= encodeAVX2Runes; i += encodeAVX2Runes {
		if !loadEncodeAVX2Chunk(inputBase, i).And(nonASCIIBits).IsZero() {
			return false
		}
	}
	for ; i < len(s); i++ {
		if uint32(s[i]) >= 0x80 {
			return false
		}
	}
	return true
}

func encodeAVX2ExtraCounts(
	chunk archsimd.Uint32x8,
	threshold2, threshold3, threshold4 archsimd.Int32x8,
) archsimd.Int32x8 {
	signed := chunk.BitsToInt32()
	return signed.GreaterEqual(threshold2).ToInt32x8().
		Add(signed.GreaterEqual(threshold3).ToInt32x8()).
		Add(signed.GreaterEqual(threshold4).ToInt32x8())
}

func encodeAVX2(s []rune, out []byte, plan encodePlan) {
	if len(out) < plan.size+15 {
		panic("utf8: AVX2 encode output buffer too small")
	}
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0
	if plan.allASCII {
		asciiShuffle := archsimd.LoadInt8x16Array(&encodeAVX2ASCII)
		for ; len(s)-i >= 4*encodeAVX2Runes; i += 4 * encodeAVX2Runes {
			packASCII8AVX2(loadEncodeAVX2Chunk(inputBase, i), asciiShuffle).
				StoreArray((*[16]uint8)(unsafe.Add(outputBase, uintptr(n))))
			packASCII8AVX2(loadEncodeAVX2Chunk(inputBase, i+encodeAVX2Runes), asciiShuffle).
				StoreArray((*[16]uint8)(unsafe.Add(outputBase, uintptr(n+8))))
			packASCII8AVX2(loadEncodeAVX2Chunk(inputBase, i+2*encodeAVX2Runes), asciiShuffle).
				StoreArray((*[16]uint8)(unsafe.Add(outputBase, uintptr(n+16))))
			packASCII8AVX2(loadEncodeAVX2Chunk(inputBase, i+3*encodeAVX2Runes), asciiShuffle).
				StoreArray((*[16]uint8)(unsafe.Add(outputBase, uintptr(n+24))))
			n += 4 * encodeAVX2Runes
		}
		for ; len(s)-i >= encodeAVX2Runes; i += encodeAVX2Runes {
			packASCII8AVX2(loadEncodeAVX2Chunk(inputBase, i), asciiShuffle).
				StoreArray((*[16]uint8)(unsafe.Add(outputBase, uintptr(n))))
			n += encodeAVX2Runes
		}
		for ; i < len(s); i++ {
			out[n] = byte(s[i])
			n++
		}
		return
	}

	for ; len(s)-i >= encodeAVX2Runes; i += encodeAVX2Runes {
		chunk := loadEncodeAVX2Chunk(inputBase, i)
		if !plan.allValid {
			chunk = normalizeEncodeAVX2Chunk(chunk)
		}
		n += encodeAVX2Chunk(outputBase, n, chunk)
	}
	for ; i < len(s); i++ {
		n += stdutf8.EncodeRune(out[n:], s[i])
	}
}

func loadEncodeAVX2Chunk(base unsafe.Pointer, i int) archsimd.Uint32x8 {
	return archsimd.LoadUint32x8Array((*[8]uint32)(unsafe.Add(base, uintptr(i)*4)))
}

func packASCII8AVX2(chunk archsimd.Uint32x8, shuffle archsimd.Int8x16) archsimd.Uint8x16 {
	return chunk.GetLo().BitsToInt32().SaturateToUint16Concat(chunk.GetHi().BitsToInt32()).
		ReshapeToUint8s().PermuteOrZero(shuffle)
}

func normalizeEncodeAVX2Chunk(chunk archsimd.Uint32x8) archsimd.Uint32x8 {
	invalid := chunk.Greater(archsimd.BroadcastUint32x8(stdutf8.MaxRune)).Or(
		chunk.And(archsimd.BroadcastUint32x8(0xfffff800)).Equal(
			archsimd.BroadcastUint32x8(0xd800),
		),
	)
	return archsimd.BroadcastUint32x8(stdutf8.RuneError).IfElse(invalid, chunk)
}

func encodeAVX2Chunk(base unsafe.Pointer, n int, chunk archsimd.Uint32x8) int {
	signed := chunk.BitsToInt32()
	threshold2 := archsimd.LoadUint32x8Array(&encodeAVX2Vectors.threshold2).BitsToInt32()
	more1Bits := signed.GreaterEqual(threshold2).ToBits()
	if more1Bits == 0 {
		shuffle := archsimd.LoadInt8x16Array(&encodeAVX2ASCII)
		packASCII8AVX2(chunk, shuffle).
			StoreArray((*[16]uint8)(unsafe.Add(base, uintptr(n))))
		return 8
	}
	threshold3 := archsimd.LoadUint32x8Array(&encodeAVX2Vectors.threshold3).BitsToInt32()
	threshold4 := archsimd.LoadUint32x8Array(&encodeAVX2Vectors.threshold4).BitsToInt32()
	more2Bits := signed.GreaterEqual(threshold3).ToBits()
	more3Bits := signed.GreaterEqual(threshold4).ToBits()
	low6 := archsimd.LoadUint32x8Array(&encodeAVX2Vectors.low6)
	continuation := archsimd.LoadUint32x8Array(&encodeAVX2Vectors.continuation)
	continuation0 := chunk.And(low6).Or(continuation)

	if more1Bits == 0xff && more2Bits == 0 {
		encoded := encodeTwoBytesAVX2(chunk, continuation0)
		return storeEncodedGroupsAVX2(encoded, base, n, 0x55, 0x55)
	}
	if more2Bits == 0 {
		twoBytes := encodeTwoBytesAVX2(chunk, continuation0)
		oneByte := chunk.And(archsimd.LoadUint32x8Array(&encodeAVX2Vectors.low7))
		encoded := oneByte.IfElse(signed.Less(threshold2), twoBytes)
		return storeSelectedGroupsAVX2(encoded, base, n, more1Bits, more2Bits, more3Bits)
	}

	continuation1 := chunk.ShiftAllRight(6).And(low6).Or(continuation)
	if more2Bits == 0xff && more3Bits == 0 {
		encoded := encodeThreeBytesAVX2(chunk, continuation0, continuation1)
		return storeEncodedGroupsAVX2(encoded, base, n, 0xaa, 0xaa)
	}
	if more3Bits == 0 {
		twoBytes := encodeTwoBytesAVX2(chunk, continuation0)
		threeBytes := encodeThreeBytesAVX2(chunk, continuation0, continuation1)
		oneByte := chunk.And(archsimd.LoadUint32x8Array(&encodeAVX2Vectors.low7))
		encoded := twoBytes.IfElse(signed.Less(threshold3), threeBytes)
		encoded = oneByte.IfElse(signed.Less(threshold2), encoded)
		return storeSelectedGroupsAVX2(encoded, base, n, more1Bits, more2Bits, more3Bits)
	}

	continuation2 := chunk.ShiftAllRight(12).And(low6).Or(continuation)
	fourBytes := encodeFourBytesAVX2(chunk, continuation0, continuation1, continuation2)
	if more3Bits == 0xff {
		fourBytes.StoreArray((*[8]uint32)(unsafe.Add(base, uintptr(n))))
		return 32
	}
	threeBytes := encodeThreeBytesAVX2(chunk, continuation0, continuation1)
	if more2Bits == 0xff {
		encoded := threeBytes.IfElse(signed.Less(threshold4), fourBytes)
		return storeSelectedGroupsAVX2(encoded, base, n, more1Bits, more2Bits, more3Bits)
	}

	twoBytes := encodeTwoBytesAVX2(chunk, continuation0)
	oneByte := chunk.And(archsimd.LoadUint32x8Array(&encodeAVX2Vectors.low7))
	encoded := threeBytes.IfElse(signed.Less(threshold4), fourBytes)
	encoded = twoBytes.IfElse(signed.Less(threshold3), encoded)
	encoded = oneByte.IfElse(signed.Less(threshold2), encoded)
	return storeSelectedGroupsAVX2(encoded, base, n, more1Bits, more2Bits, more3Bits)
}

func encodeTwoBytesAVX2(chunk, continuation0 archsimd.Uint32x8) archsimd.Uint32x8 {
	return chunk.ShiftAllRight(6).Or(archsimd.LoadUint32x8Array(&encodeAVX2Vectors.lead2)).
		Or(continuation0.ShiftAllLeft(8))
}

func encodeThreeBytesAVX2(chunk, continuation0, continuation1 archsimd.Uint32x8) archsimd.Uint32x8 {
	return chunk.ShiftAllRight(12).Or(archsimd.LoadUint32x8Array(&encodeAVX2Vectors.lead3)).
		Or(continuation1.ShiftAllLeft(8)).Or(continuation0.ShiftAllLeft(16))
}

func encodeFourBytesAVX2(
	chunk, continuation0, continuation1, continuation2 archsimd.Uint32x8,
) archsimd.Uint32x8 {
	return chunk.ShiftAllRight(18).Or(archsimd.LoadUint32x8Array(&encodeAVX2Vectors.lead4)).
		Or(continuation2.ShiftAllLeft(8)).Or(continuation1.ShiftAllLeft(16)).
		Or(continuation0.ShiftAllLeft(24))
}

func storeSelectedGroupsAVX2(
	encoded archsimd.Uint32x8,
	base unsafe.Pointer,
	n int,
	more1, more2, more3 uint8,
) int {
	selectors := encodeAVX2LengthBits[more1] +
		encodeAVX2LengthBits[more2] + encodeAVX2LengthBits[more3]
	return storeEncodedGroupsAVX2(encoded, base, n, uint8(selectors), uint8(selectors>>8))
}

func storeEncodedGroupsAVX2(
	encoded archsimd.Uint32x8,
	base unsafe.Pointer,
	n int,
	selector0, selector1 uint8,
) int {
	written0 := storeEncodedGroupAVX2(encoded.GetLo(), base, n, selector0)
	written1 := storeEncodedGroupAVX2(encoded.GetHi(), base, n+written0, selector1)
	return written0 + written1
}

func storeEncodedGroupAVX2(
	chunk archsimd.Uint32x4,
	base unsafe.Pointer,
	n int,
	selector uint8,
) int {
	shuffle := &encodeAVX2Shuffles[selector]
	chunk.ReshapeToUint8s().PermuteOrZero(
		archsimd.LoadUint8x16Array(&shuffle.indices).BitsToInt8(),
	).StoreArray((*[16]uint8)(unsafe.Add(base, uintptr(n))))
	return int(shuffle.size)
}

func buildEncodeAVX2Shuffles() [256]encodeAVX2Shuffle {
	var table [256]encodeAVX2Shuffle
	for selector := range table {
		entry := &table[selector]
		for i := range entry.indices {
			entry.indices[i] = 0xff
		}
		for lane := 0; lane < 4; lane++ {
			length := (selector>>(2*lane))&3 + 1
			for b := 0; b < length; b++ {
				entry.indices[entry.size] = uint8(lane*4 + b)
				entry.size++
			}
		}
	}
	return table
}

func buildEncodeAVX2LengthBits() [256]uint16 {
	var table [256]uint16
	for mask := range table {
		for lane := 0; lane < 8; lane++ {
			table[mask] |= uint16(mask>>lane&1) << (2 * lane)
		}
	}
	return table
}

func encodeAVX2Splat(value uint32) [8]uint32 {
	return [8]uint32{value, value, value, value, value, value, value, value}
}
