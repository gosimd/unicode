//go:build goexperiment.simd && amd64

package utf8

import (
	"math/bits"
	"simd/archsimd"
	stdutf8 "unicode/utf8"
	"unsafe"
)

const (
	decodeAVX512VectorBytes = 64
	decodeAVX512ChunkBytes  = 16
	decodeAVX512LoadAhead   = 16

	decodeAVX512DenseTwoByteContinuations  = uint64(0xaaaaaaaaaaaaaaaa)
	decodeAVX512DenseFourByteContinuations = uint64(0xeeeeeeeeeeeeeeee)
	decodeAVX512DenseThreeByte48Mask       = uint64(0x0000db6db6db6db6)
	decodeAVX512Low48Mask                  = uint64(0x0000ffffffffffff)
)

var decodeAVX512Tables = struct {
	leadPayloadMask  [16]uint8
	leadShift        [16]uint8
	dense2LeadMask   [16]uint16
	dense2LastMask   [16]uint16
	dense4LastMask   [16]uint32
	dense4MiddleMask [16]uint32
	dense4HighMask   [16]uint32
	dense4FirstMask  [16]uint32
	dense3LeadMask   [16]uint8
	dense3LastMask   [16]uint8
}{
	leadPayloadMask: [16]uint8{
		0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f, 0x7f,
		0x00, 0x00, 0x00, 0x00, 0x1f, 0x1f, 0x0f, 0x07,
	},
	leadShift: [16]uint8{
		0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 6, 6, 12, 18,
	},
	dense2LeadMask: [16]uint16{
		0x001f, 0x001f, 0x001f, 0x001f, 0x001f, 0x001f, 0x001f, 0x001f,
		0x001f, 0x001f, 0x001f, 0x001f, 0x001f, 0x001f, 0x001f, 0x001f,
	},
	dense2LastMask: [16]uint16{
		0x003f, 0x003f, 0x003f, 0x003f, 0x003f, 0x003f, 0x003f, 0x003f,
		0x003f, 0x003f, 0x003f, 0x003f, 0x003f, 0x003f, 0x003f, 0x003f,
	},
	dense4LastMask: [16]uint32{
		0x0000003f, 0x0000003f, 0x0000003f, 0x0000003f,
		0x0000003f, 0x0000003f, 0x0000003f, 0x0000003f,
		0x0000003f, 0x0000003f, 0x0000003f, 0x0000003f,
		0x0000003f, 0x0000003f, 0x0000003f, 0x0000003f,
	},
	dense4MiddleMask: [16]uint32{
		0x00000fc0, 0x00000fc0, 0x00000fc0, 0x00000fc0,
		0x00000fc0, 0x00000fc0, 0x00000fc0, 0x00000fc0,
		0x00000fc0, 0x00000fc0, 0x00000fc0, 0x00000fc0,
		0x00000fc0, 0x00000fc0, 0x00000fc0, 0x00000fc0,
	},
	dense4HighMask: [16]uint32{
		0x0003f000, 0x0003f000, 0x0003f000, 0x0003f000,
		0x0003f000, 0x0003f000, 0x0003f000, 0x0003f000,
		0x0003f000, 0x0003f000, 0x0003f000, 0x0003f000,
		0x0003f000, 0x0003f000, 0x0003f000, 0x0003f000,
	},
	dense4FirstMask: [16]uint32{
		0x001c0000, 0x001c0000, 0x001c0000, 0x001c0000,
		0x001c0000, 0x001c0000, 0x001c0000, 0x001c0000,
		0x001c0000, 0x001c0000, 0x001c0000, 0x001c0000,
		0x001c0000, 0x001c0000, 0x001c0000, 0x001c0000,
	},
	dense3LeadMask: [16]uint8{
		0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f,
		0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f, 0x0f,
	},
	dense3LastMask: [16]uint8{
		0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f,
		0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f, 0x3f,
	},
}

var decodeAVX512Dense3Shuffle = buildDecodeAVX512Dense3Shuffle()

func buildDecodeAVX512Dense3Shuffle() (tables [3][3][16]uint8) {
	for byteInRune := range tables {
		for source := range tables[byteInRune] {
			for lane := range tables[byteInRune][source] {
				tables[byteInRune][source][lane] = 0xff
			}
		}
	}
	for lane := range 16 {
		for byteInRune := range 3 {
			position := 3*lane + byteInRune
			source := position / decodeAVX512ChunkBytes
			tables[byteInRune][source][lane] = uint8(position % decodeAVX512ChunkBytes)
		}
	}
	return tables
}

// decodeValidAVX512 decodes already-validated UTF-8 into caller-owned output.
// out must contain exactly outputRunes writable lanes. The AVX-512 path treats
// every non-continuation byte as a rune start, decodes those starts in parallel,
// and compacts the resulting UTF-32 lanes with VPCOMPRESSD.
func decodeValidAVX512(input []byte, out []rune, outputRunes int) []rune {
	if len(out) < outputRunes {
		panic("utf8: decode output buffer too small")
	}

	inputBase := unsafe.Pointer(unsafe.SliceData(input))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	i, n := 0, 0
	zero64 := archsimd.Int8x64{}
	continuationThreshold64 := archsimd.BroadcastInt8x64(-64)
	highNibbleMask := archsimd.BroadcastUint8x16(0x0f)
	low6 := archsimd.BroadcastUint8x16(0x3f)
	shift6 := archsimd.BroadcastUint32x16(6)
	shift12 := archsimd.BroadcastUint32x16(12)
	shift18 := archsimd.BroadcastUint32x16(18)
	leadPayloadMasks := archsimd.LoadUint8x16Array(&decodeAVX512Tables.leadPayloadMask)
	leadShifts := archsimd.LoadUint8x16Array(&decodeAVX512Tables.leadShift)

	for len(input)-i >= decodeAVX512VectorBytes {
		block := archsimd.LoadUint8x64(input[i:])
		if block.BitsToInt8().Less(zero64).ToBits() == 0 {
			storeASCIIBytes64AVX512(block, unsafe.Add(outputBase, uintptr(n)*4))
			i += decodeAVX512VectorBytes
			n += decodeAVX512VectorBytes
			continue
		}
		continuationBits64 := block.BitsToInt8().Less(continuationThreshold64).ToBits()
		switch continuationBits64 {
		case decodeAVX512DenseTwoByteContinuations:
			lo, hi := block.GetLo().AsUint16x16(), block.GetHi().AsUint16x16()
			leadMask := archsimd.LoadUint16x16Array(&decodeAVX512Tables.dense2LeadMask)
			lastMask := archsimd.LoadUint16x16Array(&decodeAVX512Tables.dense2LastMask)
			decodedLo := lo.And(leadMask).ShiftAllLeft(6).
				Or(lo.ShiftAllRight(8).And(lastMask))
			decodedHi := hi.And(leadMask).ShiftAllLeft(6).
				Or(hi.ShiftAllRight(8).And(lastMask))
			decodedLo.ExtendToUint32().BitsToInt32().
				StoreArray((*[16]int32)(unsafe.Add(outputBase, uintptr(n)*4)))
			decodedHi.ExtendToUint32().BitsToInt32().StoreArray(
				(*[16]int32)(unsafe.Add(outputBase, uintptr(n+16)*4)),
			)
			i += decodeAVX512VectorBytes
			n += 32
			continue
		case decodeAVX512DenseFourByteContinuations:
			words := block.AsUint32x16()
			last := words.ShiftAllRight(24).And(
				archsimd.LoadUint32x16Array(&decodeAVX512Tables.dense4LastMask),
			)
			middle := words.ShiftAllRight(10).And(
				archsimd.LoadUint32x16Array(&decodeAVX512Tables.dense4MiddleMask),
			)
			high := words.ShiftAllLeft(4).And(
				archsimd.LoadUint32x16Array(&decodeAVX512Tables.dense4HighMask),
			)
			first := words.ShiftAllLeft(18).And(
				archsimd.LoadUint32x16Array(&decodeAVX512Tables.dense4FirstMask),
			)
			last.Or(middle).Or(high).Or(first).BitsToInt32().
				StoreArray((*[16]int32)(unsafe.Add(outputBase, uintptr(n)*4)))
			i += decodeAVX512VectorBytes
			n += 16
			continue
		}
		if continuationBits64&decodeAVX512Low48Mask == decodeAVX512DenseThreeByte48Mask {
			consumed, produced := decodeDenseThreeByteAVX512(
				inputBase,
				len(input),
				i,
				unsafe.Add(outputBase, uintptr(n)*4),
			)
			i += consumed
			n += produced
			continue
		}
		// The general decoder loads the following 16-byte vector even though
		// only its first three bytes are semantically needed as lookahead.
		if len(input)-i < decodeAVX512VectorBytes+decodeAVX512LoadAhead {
			break
		}

		end := i + decodeAVX512VectorBytes
		chunk := archsimd.LoadUint8x16Array(
			(*[decodeAVX512ChunkBytes]uint8)(unsafe.Add(inputBase, uintptr(i))),
		)
		for i < end {
			next := archsimd.LoadUint8x16Array(
				(*[decodeAVX512ChunkBytes]uint8)(unsafe.Add(
					inputBase, uintptr(i+decodeAVX512ChunkBytes),
				)),
			)

			runeStartBits := uint16(^continuationBits64)
			produced := bits.OnesCount16(runeStartBits)

			forward1 := next.ConcatShiftBytesRight(chunk, 1)
			forward2 := next.ConcatShiftBytesRight(chunk, 2)
			forward3 := next.ConcatShiftBytesRight(chunk, 3)
			highNibbles := chunk.AsUint16x8().ShiftAllRight(4).AsUint8x16().
				And(highNibbleMask).BitsToInt8()
			leadPayload := chunk.And(leadPayloadMasks.PermuteOrZero(highNibbles))
			shifts := leadShifts.PermuteOrZero(highNibbles).ExtendToUint32()
			decoded := leadPayload.ExtendToUint32().ShiftLeft(shifts).
				Or(forward1.And(low6).ExtendToUint32().ShiftLeft(shifts.Sub(shift6))).
				Or(forward2.And(low6).ExtendToUint32().ShiftLeft(shifts.Sub(shift12))).
				Or(forward3.And(low6).ExtendToUint32().ShiftLeft(shifts.Sub(shift18)))

			packed := decoded.Compress(archsimd.Mask32x16FromBits(runeStartBits))
			storeMask := archsimd.Mask32x16FromBits(
				uint16((uint32(1) << produced) - 1),
			)
			packed.BitsToInt32().StoreArrayMasked(
				(*[16]int32)(unsafe.Add(outputBase, uintptr(n)*4)),
				storeMask,
			)
			i += decodeAVX512ChunkBytes
			n += produced
			chunk = next
			continuationBits64 >>= decodeAVX512ChunkBytes
		}
	}

	// A rune beginning in the last SIMD chunk may consume up to three bytes at
	// the scalar boundary. It has already been emitted by the lookahead decoder.
	for i < len(input) && input[i]&0xc0 == 0x80 {
		i++
	}
	for i < len(input) {
		r, size := stdutf8.DecodeRune(input[i:])
		out[n] = r
		i += size
		n++
	}
	return out[:n]
}

//go:noinline
func decodeDenseThreeByteAVX512(
	inputBase unsafe.Pointer,
	inputLen int,
	inputOffset int,
	output unsafe.Pointer,
) (consumed, produced int) {
	continuationThreshold := archsimd.BroadcastInt8x64(-64)
	low4 := archsimd.LoadUint8x16Array(&decodeAVX512Tables.dense3LeadMask)
	low6 := archsimd.LoadUint8x16Array(&decodeAVX512Tables.dense3LastMask)
	i := inputOffset
	for inputLen-i >= decodeAVX512VectorBytes {
		block := archsimd.LoadUint8x64Array(
			(*[64]uint8)(unsafe.Add(inputBase, uintptr(i))),
		)
		continuations := block.BitsToInt8().Less(continuationThreshold).ToBits()
		if continuations&decodeAVX512Low48Mask != decodeAVX512DenseThreeByte48Mask {
			break
		}

		chunk0 := archsimd.LoadUint8x16Array(
			(*[16]uint8)(unsafe.Add(inputBase, uintptr(i))),
		)
		chunk1 := archsimd.LoadUint8x16Array(
			(*[16]uint8)(unsafe.Add(inputBase, uintptr(i+16))),
		)
		chunk2 := archsimd.LoadUint8x16Array(
			(*[16]uint8)(unsafe.Add(inputBase, uintptr(i+32))),
		)
		lead := chunk0.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX512Dense3Shuffle[0][0]).BitsToInt8(),
		).Or(chunk1.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX512Dense3Shuffle[0][1]).BitsToInt8(),
		)).Or(chunk2.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX512Dense3Shuffle[0][2]).BitsToInt8(),
		))
		middle := chunk0.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX512Dense3Shuffle[1][0]).BitsToInt8(),
		).Or(chunk1.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX512Dense3Shuffle[1][1]).BitsToInt8(),
		)).Or(chunk2.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX512Dense3Shuffle[1][2]).BitsToInt8(),
		))
		last := chunk0.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX512Dense3Shuffle[2][0]).BitsToInt8(),
		).Or(chunk1.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX512Dense3Shuffle[2][1]).BitsToInt8(),
		)).Or(chunk2.PermuteOrZero(
			archsimd.LoadUint8x16Array(&decodeAVX512Dense3Shuffle[2][2]).BitsToInt8(),
		))
		decoded := lead.And(low4).ExtendToUint32().ShiftAllLeft(12).
			Or(middle.And(low6).ExtendToUint32().ShiftAllLeft(6)).
			Or(last.And(low6).ExtendToUint32())
		decoded.BitsToInt32().StoreArray(
			(*[16]int32)(unsafe.Add(output, uintptr(produced)*4)),
		)
		i += 48
		produced += 16
	}
	return i - inputOffset, produced
}

func storeASCIIBytes64AVX512(block archsimd.Uint8x64, output unsafe.Pointer) {
	lo, hi := block.GetLo(), block.GetHi()
	lo.GetLo().ExtendToUint32().BitsToInt32().
		StoreArray((*[16]int32)(output))
	lo.GetHi().ExtendToUint32().BitsToInt32().
		StoreArray((*[16]int32)(unsafe.Add(output, 16*4)))
	hi.GetLo().ExtendToUint32().BitsToInt32().
		StoreArray((*[16]int32)(unsafe.Add(output, 32*4)))
	hi.GetHi().ExtendToUint32().BitsToInt32().
		StoreArray((*[16]int32)(unsafe.Add(output, 48*4)))
}
