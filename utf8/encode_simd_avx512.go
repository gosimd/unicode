//go:build goexperiment.simd && amd64

package utf8

import (
	"math/bits"
	"simd/archsimd"
	stdutf8 "unicode/utf8"
	"unsafe"
)

type encodeAVX512Shuffle struct {
	size    uint8
	indices [16]uint8
}

var (
	encodeAVX512Shuffles  = buildEncodeAVX512Shuffles()
	encodeAVX512Selectors = buildEncodeAVX512Selectors()
)

func planEncodeAVX512(s []rune) encodePlan {
	if allASCIIEncodeAVX512(s) {
		return encodePlan{size: len(s), allASCII: true, allValid: true}
	}

	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	threshold2 := archsimd.BroadcastUint32x16(0x80)
	threshold3 := archsimd.BroadcastUint32x16(0x800)
	threshold4 := archsimd.BroadcastUint32x16(0x10000)
	maxRune := archsimd.BroadcastUint32x16(stdutf8.MaxRune)
	surrogateMin := archsimd.BroadcastUint32x16(0xd800)
	surrogateMax := archsimd.BroadcastUint32x16(0xdfff)
	replacement := archsimd.BroadcastUint32x16(stdutf8.RuneError)

	size := len(s)
	allASCII := true
	allValid := true
	i := 0
	for ; len(s)-i >= 16; i += 16 {
		chunk := archsimd.LoadUint32x16Array(
			(*[16]uint32)(unsafe.Add(inputBase, uintptr(i)*4)),
		)
		invalid := chunk.Greater(maxRune).Or(
			chunk.GreaterEqual(surrogateMin).And(chunk.LessEqual(surrogateMax)),
		)
		allValid = allValid && invalid.ToBits() == 0
		chunk = replacement.IfElse(invalid, chunk)
		more1 := chunk.GreaterEqual(threshold2).ToBits()
		more2 := chunk.GreaterEqual(threshold3).ToBits()
		more3 := chunk.GreaterEqual(threshold4).ToBits()
		size += bits.OnesCount16(more1) + bits.OnesCount16(more2) + bits.OnesCount16(more3)
		allASCII = allASCII && more1 == 0
	}
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
		allASCII = allASCII && r < 0x80
	}
	return encodePlan{size: size, allASCII: allASCII, allValid: allValid}
}

func allASCIIEncodeAVX512(s []rune) bool {
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	threshold := archsimd.BroadcastUint32x16(0x80)
	i := 0
	for ; len(s)-i >= 64; i += 64 {
		chunk0 := archsimd.LoadUint32x16Array(
			(*[16]uint32)(unsafe.Add(inputBase, uintptr(i)*4)),
		)
		chunk1 := archsimd.LoadUint32x16Array(
			(*[16]uint32)(unsafe.Add(inputBase, uintptr(i+16)*4)),
		)
		chunk2 := archsimd.LoadUint32x16Array(
			(*[16]uint32)(unsafe.Add(inputBase, uintptr(i+32)*4)),
		)
		chunk3 := archsimd.LoadUint32x16Array(
			(*[16]uint32)(unsafe.Add(inputBase, uintptr(i+48)*4)),
		)
		if chunk0.Or(chunk1).Or(chunk2.Or(chunk3)).Less(threshold).ToBits() != 0xffff {
			return false
		}
	}
	for ; len(s)-i >= 16; i += 16 {
		chunk := archsimd.LoadUint32x16Array(
			(*[16]uint32)(unsafe.Add(inputBase, uintptr(i)*4)),
		)
		if chunk.Less(threshold).ToBits() != 0xffff {
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

func encodeAVX512(s []rune, out []byte, plan encodePlan) {
	if len(out) < plan.size+15 {
		panic("utf8: AVX-512 encode output buffer too small")
	}

	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0
	if plan.allASCII {
		for ; len(s)-i >= 64; i += 64 {
			chunk0 := archsimd.LoadUint32x16Array(
				(*[16]uint32)(unsafe.Add(inputBase, uintptr(i)*4)),
			)
			chunk1 := archsimd.LoadUint32x16Array(
				(*[16]uint32)(unsafe.Add(inputBase, uintptr(i+16)*4)),
			)
			chunk2 := archsimd.LoadUint32x16Array(
				(*[16]uint32)(unsafe.Add(inputBase, uintptr(i+32)*4)),
			)
			chunk3 := archsimd.LoadUint32x16Array(
				(*[16]uint32)(unsafe.Add(inputBase, uintptr(i+48)*4)),
			)
			chunk0.TruncToUint8().StoreArray(
				(*[16]uint8)(unsafe.Add(outputBase, uintptr(n))),
			)
			chunk1.TruncToUint8().StoreArray(
				(*[16]uint8)(unsafe.Add(outputBase, uintptr(n+16))),
			)
			chunk2.TruncToUint8().StoreArray(
				(*[16]uint8)(unsafe.Add(outputBase, uintptr(n+32))),
			)
			chunk3.TruncToUint8().StoreArray(
				(*[16]uint8)(unsafe.Add(outputBase, uintptr(n+48))),
			)
			n += 64
		}
		for ; len(s)-i >= 16; i += 16 {
			chunk := archsimd.LoadUint32x16Array(
				(*[16]uint32)(unsafe.Add(inputBase, uintptr(i)*4)),
			)
			chunk.TruncToUint8().StoreArray(
				(*[16]uint8)(unsafe.Add(outputBase, uintptr(n))),
			)
			n += 16
		}
		for ; i < len(s); i++ {
			out[n] = byte(s[i])
			n++
		}
		return
	}
	threshold2 := archsimd.BroadcastUint32x16(0x80)
	threshold3 := archsimd.BroadcastUint32x16(0x800)
	threshold4 := archsimd.BroadcastUint32x16(0x10000)
	maxRune := archsimd.BroadcastUint32x16(stdutf8.MaxRune)
	surrogateMin := archsimd.BroadcastUint32x16(0xd800)
	surrogateMax := archsimd.BroadcastUint32x16(0xdfff)
	replacement := archsimd.BroadcastUint32x16(stdutf8.RuneError)
	low6 := archsimd.BroadcastUint32x16(0x3f)
	continuation := archsimd.BroadcastUint32x16(0x80)
	low7 := archsimd.BroadcastUint32x16(0x7f)
	lead2 := archsimd.BroadcastUint32x16(0xc0)
	lead3 := archsimd.BroadcastUint32x16(0xe0)
	lead4 := archsimd.BroadcastUint32x16(0xf0)

	for ; len(s)-i >= 16; i += 16 {
		chunk := archsimd.LoadUint32x16Array(
			(*[16]uint32)(unsafe.Add(inputBase, uintptr(i)*4)),
		)
		if !plan.allValid {
			invalid := chunk.Greater(maxRune).Or(
				chunk.GreaterEqual(surrogateMin).And(chunk.LessEqual(surrogateMax)),
			)
			chunk = replacement.IfElse(invalid, chunk)
		}
		more1 := chunk.GreaterEqual(threshold2)
		more2 := chunk.GreaterEqual(threshold3)
		more3 := chunk.GreaterEqual(threshold4)
		more1Bits := more1.ToBits()
		more2Bits := more2.ToBits()
		more3Bits := more3.ToBits()

		if more1Bits == 0 {
			chunk.TruncToUint8().StoreArray(
				(*[16]uint8)(unsafe.Add(outputBase, uintptr(n))),
			)
			n += 16
			continue
		}

		continuation0 := chunk.And(low6).Or(continuation)
		twoBytes := chunk.ShiftAllRight(6).Or(lead2).
			Or(continuation0.ShiftAllLeft(8))
		if more1Bits == 0xffff && more2Bits == 0 {
			twoBytes.TruncToUint16().StoreArray(
				(*[16]uint16)(unsafe.Add(outputBase, uintptr(n))),
			)
			n += 32
			continue
		}

		continuation1 := chunk.ShiftAllRight(6).And(low6).Or(continuation)
		threeBytes := chunk.ShiftAllRight(12).Or(lead3).
			Or(continuation1.ShiftAllLeft(8)).Or(continuation0.ShiftAllLeft(16))
		if more2Bits == 0xffff && more3Bits == 0 {
			n += storeEncodedGroupsAVX512(
				threeBytes.AsUint8x64(), outputBase, n,
				0xaa, 0xaa, 0xaa, 0xaa,
			)
			continue
		}

		continuation2 := chunk.ShiftAllRight(12).And(low6).Or(continuation)
		fourBytes := chunk.ShiftAllRight(18).Or(lead4).
			Or(continuation2.ShiftAllLeft(8)).Or(continuation1.ShiftAllLeft(16)).
			Or(continuation0.ShiftAllLeft(24))
		if more3Bits == 0xffff {
			fourBytes.StoreArray((*[16]uint32)(unsafe.Add(outputBase, uintptr(n))))
			n += 64
			continue
		}

		oneByte := chunk.And(low7)
		encoded := threeBytes.IfElse(chunk.Less(threshold4), fourBytes)
		encoded = twoBytes.IfElse(chunk.Less(threshold3), encoded)
		encoded = oneByte.IfElse(chunk.Less(threshold2), encoded)
		n += storeEncodedGroupsAVX512(
			encoded.AsUint8x64(), outputBase, n,
			encodeSelectorAVX512(more1Bits, more2Bits, more3Bits, 0),
			encodeSelectorAVX512(more1Bits, more2Bits, more3Bits, 4),
			encodeSelectorAVX512(more1Bits, more2Bits, more3Bits, 8),
			encodeSelectorAVX512(more1Bits, more2Bits, more3Bits, 12),
		)
	}
	for ; i < len(s); i++ {
		n += stdutf8.EncodeRune(out[n:], s[i])
	}
}

func storeEncodedGroupsAVX512(
	encoded archsimd.Uint8x64,
	outputBase unsafe.Pointer,
	n int,
	selector0, selector1, selector2, selector3 uint8,
) int {
	lo, hi := encoded.GetLo(), encoded.GetHi()
	written := 0
	written += storeEncodedGroupAVX512(lo.GetLo(), outputBase, n+written, selector0)
	written += storeEncodedGroupAVX512(lo.GetHi(), outputBase, n+written, selector1)
	written += storeEncodedGroupAVX512(hi.GetLo(), outputBase, n+written, selector2)
	written += storeEncodedGroupAVX512(hi.GetHi(), outputBase, n+written, selector3)
	return written
}

func storeEncodedGroupAVX512(chunk archsimd.Uint8x16, outputBase unsafe.Pointer, n int, selector uint8) int {
	shuffle := &encodeAVX512Shuffles[selector]
	chunk.PermuteOrZero(archsimd.LoadUint8x16Array(&shuffle.indices).BitsToInt8()).
		StoreArray((*[16]uint8)(unsafe.Add(outputBase, uintptr(n))))
	return int(shuffle.size)
}

func encodeSelectorAVX512(more1, more2, more3 uint16, shift uint) uint8 {
	index := (more1 >> shift & 0x0f) |
		(more2>>shift&0x0f)<<4 |
		(more3>>shift&0x0f)<<8
	return encodeAVX512Selectors[index]
}

func buildEncodeAVX512Shuffles() [256]encodeAVX512Shuffle {
	var table [256]encodeAVX512Shuffle
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

func buildEncodeAVX512Selectors() [4096]uint8 {
	var table [4096]uint8
	for index := range table {
		more1 := index & 0x0f
		more2 := index >> 4 & 0x0f
		more3 := index >> 8 & 0x0f
		selector := 0
		for lane := 0; lane < 4; lane++ {
			lengthCode := ((more1 >> lane) & 1) +
				((more2 >> lane) & 1) +
				((more3 >> lane) & 1)
			selector |= lengthCode << (2 * lane)
		}
		table[index] = uint8(selector)
	}
	return table
}
