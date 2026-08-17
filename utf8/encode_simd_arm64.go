//go:build goexperiment.simd && arm64

package utf8

import (
	"simd/archsimd"
	stdutf8 "unicode/utf8"
	"unsafe"
)

const encodeSIMDRunes = 4

type encodePlan struct {
	size     int
	allASCII bool
}

type encodeShuffle struct {
	size    uint8
	indices [16]uint8
}

// The low two bits for every rune select a one-, two-, three-, or four-byte
// encoding. Each table row compacts four little-endian uint32 lanes into one
// contiguous UTF-8 byte sequence; 0xff makes LookupOrZero clear unused bytes.
var encodeShuffles = buildEncodeShuffles()

var encodeVectors = struct {
	threshold2, threshold3, threshold4  [4]uint32
	maxRune, surrogateMin, surrogateMax [4]uint32
	replacement, low6, continuation     [4]uint32
	low7, lead2, lead3, lead4           [4]uint32
	max2, max3, weights                 [4]uint32
}{
	threshold2:   [4]uint32{0x80, 0x80, 0x80, 0x80},
	threshold3:   [4]uint32{0x800, 0x800, 0x800, 0x800},
	threshold4:   [4]uint32{0x10000, 0x10000, 0x10000, 0x10000},
	maxRune:      [4]uint32{stdutf8.MaxRune, stdutf8.MaxRune, stdutf8.MaxRune, stdutf8.MaxRune},
	surrogateMin: [4]uint32{0xd800, 0xd800, 0xd800, 0xd800},
	surrogateMax: [4]uint32{0xdfff, 0xdfff, 0xdfff, 0xdfff},
	replacement:  [4]uint32{stdutf8.RuneError, stdutf8.RuneError, stdutf8.RuneError, stdutf8.RuneError},
	low6:         [4]uint32{0x3f, 0x3f, 0x3f, 0x3f},
	continuation: [4]uint32{0x80, 0x80, 0x80, 0x80},
	low7:         [4]uint32{0x7f, 0x7f, 0x7f, 0x7f},
	lead2:        [4]uint32{0xc0, 0xc0, 0xc0, 0xc0},
	lead3:        [4]uint32{0xe0, 0xe0, 0xe0, 0xe0},
	lead4:        [4]uint32{0xf0, 0xf0, 0xf0, 0xf0},
	max2:         [4]uint32{0x7ff, 0x7ff, 0x7ff, 0x7ff},
	max3:         [4]uint32{0xffff, 0xffff, 0xffff, 0xffff},
	weights:      [4]uint32{1, 4, 16, 64},
}

// Encode returns the UTF-8 encoding of s. Runes outside the valid Unicode
// range are replaced by RuneError, as in the language conversion string(s).
func Encode(s []rune) string {
	if len(s) == 0 {
		return ""
	}

	plan := planEncodeSIMD(s)
	// encodeSIMDChunk always performs one full 16-byte store. Four input runes
	// produce at least four output bytes, so 15 padding bytes are sufficient
	// even for the final chunk and are excluded from the returned string.
	out := make([]byte, plan.size+15)
	encodeSIMD(s, out, plan)
	return unsafe.String(unsafe.SliceData(out), plan.size)
}

func planEncodeSIMD(s []rune) encodePlan {
	threshold2 := archsimd.LoadUint32x4Array(&encodeVectors.threshold2)
	threshold3 := archsimd.LoadUint32x4Array(&encodeVectors.threshold3)
	threshold4 := archsimd.LoadUint32x4Array(&encodeVectors.threshold4)
	maxRune := archsimd.LoadUint32x4Array(&encodeVectors.maxRune)
	surrogateMin := archsimd.LoadUint32x4Array(&encodeVectors.surrogateMin)
	surrogateMax := archsimd.LoadUint32x4Array(&encodeVectors.surrogateMax)
	replacement := archsimd.LoadUint32x4Array(&encodeVectors.replacement)

	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	var extra, maximum archsimd.Uint32x4
	i := 0
	for ; len(s)-i >= encodeSIMDRunes; i += encodeSIMDRunes {
		chunk := archsimd.LoadUint32x4Array((*[4]uint32)(unsafe.Add(inputBase, uintptr(i)*4)))
		invalid := chunk.Greater(maxRune).Or(
			chunk.GreaterEqual(surrogateMin).And(chunk.LessEqual(surrogateMax)),
		)
		chunk = replacement.IfElse(invalid, chunk)

		extra = extra.Add(chunk.GreaterEqual(threshold2).ToInt32x4().Neg().ToBits())
		extra = extra.Add(chunk.GreaterEqual(threshold3).ToInt32x4().Neg().ToBits())
		extra = extra.Add(chunk.GreaterEqual(threshold4).ToInt32x4().Neg().ToBits())
		maximum = maximum.Max(chunk)
	}

	size := len(s) + int(extra.ReduceSum())
	maxValue := maximum.ReduceMax()
	for ; i < len(s); i++ {
		r := s[i]
		if r < 0 || r > stdutf8.MaxRune || r >= 0xd800 && r <= 0xdfff {
			r = stdutf8.RuneError
		}
		switch {
		case r >= 0x10000:
			size += 3
		case r >= 0x800:
			size += 2
		case r >= 0x80:
			size++
		}
		if uint32(r) > maxValue {
			maxValue = uint32(r)
		}
	}
	return encodePlan{size: size, allASCII: maxValue < 0x80}
}

func encodeSIMD(s []rune, out []byte, plan encodePlan) {
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0

	if plan.allASCII {
		for ; len(s)-i >= 16; i += 16 {
			chunk0 := loadEncodeChunk(inputBase, i)
			chunk1 := loadEncodeChunk(inputBase, i+4)
			chunk2 := loadEncodeChunk(inputBase, i+8)
			chunk3 := loadEncodeChunk(inputBase, i+12)
			storeASCII16(outputBase, n, chunk0, chunk1, chunk2, chunk3)
			n += 16
		}
		for ; i < len(s); i++ {
			out[n] = byte(s[i])
			n++
		}
		return
	}

	for ; len(s)-i >= 16; i += 16 {
		chunk0 := loadEncodeChunk(inputBase, i)
		chunk1 := loadEncodeChunk(inputBase, i+4)
		chunk2 := loadEncodeChunk(inputBase, i+8)
		chunk3 := loadEncodeChunk(inputBase, i+12)
		if chunk0.Or(chunk1).Or(chunk2).Or(chunk3).ReduceMax() < 0x80 {
			storeASCII16(outputBase, n, chunk0, chunk1, chunk2, chunk3)
			n += 16
			continue
		}
		n += encodeSIMDChunk(outputBase, n, chunk0)
		n += encodeSIMDChunk(outputBase, n, chunk1)
		n += encodeSIMDChunk(outputBase, n, chunk2)
		n += encodeSIMDChunk(outputBase, n, chunk3)
	}
	for ; len(s)-i >= encodeSIMDRunes; i += encodeSIMDRunes {
		n += encodeSIMDChunk(outputBase, n, loadEncodeChunk(inputBase, i))
	}
	for ; i < len(s); i++ {
		n += stdutf8.EncodeRune(out[n:], s[i])
	}
}

func loadEncodeChunk(base unsafe.Pointer, i int) archsimd.Uint32x4 {
	return archsimd.LoadUint32x4Array((*[4]uint32)(unsafe.Add(base, uintptr(i)*4)))
}

func storeASCII16(base unsafe.Pointer, n int, chunk0, chunk1, chunk2, chunk3 archsimd.Uint32x4) {
	packed0 := chunk0.TruncToUint16().ReshapeToUint64s().
		InterleaveLo(chunk1.TruncToUint16().ReshapeToUint64s()).
		ReshapeToUint16s().TruncToUint8()
	packed1 := chunk2.TruncToUint16().ReshapeToUint64s().
		InterleaveLo(chunk3.TruncToUint16().ReshapeToUint64s()).
		ReshapeToUint16s().TruncToUint8()
	packed0.ReshapeToUint64s().InterleaveLo(packed1.ReshapeToUint64s()).
		ReshapeToUint8s().StoreArray((*[16]uint8)(unsafe.Add(base, uintptr(n))))
}

func encodeSIMDChunk(base unsafe.Pointer, n int, chunk archsimd.Uint32x4) int {
	maximum := chunk.ReduceMax()
	if maximum < 0x80 {
		chunk.TruncToUint16().TruncToUint8().
			StoreArray((*[16]uint8)(unsafe.Add(base, uintptr(n))))
		return 4
	}
	// A four-rune vector that mixes ASCII and wider encodings is cheaper to
	// encode scalarly than to construct and compact four candidate sequences.
	// Dense non-ASCII vectors continue through the NEON table path below.
	if chunk.ReduceMin() < 0x80 {
		var runes [4]int32
		chunk.BitsToInt32().StoreArray(&runes)
		output := unsafe.Slice((*byte)(unsafe.Add(base, uintptr(n))), 16)
		written := 0
		for _, r := range runes {
			written += stdutf8.EncodeRune(output[written:], rune(r))
		}
		return written
	}

	maxRune := archsimd.LoadUint32x4Array(&encodeVectors.maxRune)
	surrogateMin := archsimd.LoadUint32x4Array(&encodeVectors.surrogateMin)
	surrogateMax := archsimd.LoadUint32x4Array(&encodeVectors.surrogateMax)
	replacement := archsimd.LoadUint32x4Array(&encodeVectors.replacement)
	invalid := chunk.Greater(maxRune).Or(
		chunk.GreaterEqual(surrogateMin).And(chunk.LessEqual(surrogateMax)),
	)
	chunk = replacement.IfElse(invalid, chunk)

	low6 := archsimd.LoadUint32x4Array(&encodeVectors.low6)
	continuation := archsimd.LoadUint32x4Array(&encodeVectors.continuation)
	continuation0 := chunk.And(low6).Or(continuation)
	continuation1 := chunk.ShiftAllRight(6).And(low6).Or(continuation)
	continuation2 := chunk.ShiftAllRight(12).And(low6).Or(continuation)

	oneByte := chunk.And(archsimd.LoadUint32x4Array(&encodeVectors.low7))
	twoBytes := chunk.ShiftAllRight(6).Or(archsimd.LoadUint32x4Array(&encodeVectors.lead2)).
		Or(continuation0.ShiftAllLeft(8))
	threeBytes := chunk.ShiftAllRight(12).Or(archsimd.LoadUint32x4Array(&encodeVectors.lead3)).
		Or(continuation1.ShiftAllLeft(8)).Or(continuation0.ShiftAllLeft(16))
	fourBytes := chunk.ShiftAllRight(18).Or(archsimd.LoadUint32x4Array(&encodeVectors.lead4)).
		Or(continuation2.ShiftAllLeft(8)).Or(continuation1.ShiftAllLeft(16)).
		Or(continuation0.ShiftAllLeft(24))

	encoded := threeBytes.IfElse(chunk.LessEqual(archsimd.LoadUint32x4Array(&encodeVectors.max3)), fourBytes)
	encoded = twoBytes.IfElse(chunk.LessEqual(archsimd.LoadUint32x4Array(&encodeVectors.max2)), encoded)
	encoded = oneByte.IfElse(chunk.LessEqual(archsimd.LoadUint32x4Array(&encodeVectors.low7)), encoded)

	lengthCodes := chunk.GreaterEqual(archsimd.LoadUint32x4Array(&encodeVectors.threshold2)).ToInt32x4().Neg().ToBits().
		Add(chunk.GreaterEqual(archsimd.LoadUint32x4Array(&encodeVectors.threshold3)).ToInt32x4().Neg().ToBits()).
		Add(chunk.GreaterEqual(archsimd.LoadUint32x4Array(&encodeVectors.threshold4)).ToInt32x4().Neg().ToBits())
	weights := archsimd.LoadUint32x4Array(&encodeVectors.weights)
	selector := lengthCodes.Mul(weights).ReduceSum()
	shuffle := &encodeShuffles[selector]
	indices := archsimd.LoadUint8x16Array(&shuffle.indices)
	encoded.ReshapeToUint8s().LookupOrZero(indices).
		StoreArray((*[16]uint8)(unsafe.Add(base, uintptr(n))))
	return int(shuffle.size)
}

func buildEncodeShuffles() [256]encodeShuffle {
	var table [256]encodeShuffle
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
