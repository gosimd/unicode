//go:build goexperiment.simd && (amd64 || arm64)

package utf16

import (
	"testing"
	stdutf16 "unicode/utf16"
)

var benchRunesSink []rune
var benchCodeUnitsSink []uint16

func BenchmarkDecode(b *testing.B) {
	for _, input := range utf16BenchmarkInputs() {
		b.Run(input.name, func(b *testing.B) {
			b.Run("stdlib_decode", func(b *testing.B) {
				b.SetBytes(int64(len(input.data) * 2))
				for b.Loop() {
					benchRunesSink = stdutf16.Decode(input.data)
				}
			})
			b.Run("stdlib_core", func(b *testing.B) {
				out := make([]rune, 0, len(input.data))
				b.SetBytes(int64(len(input.data) * 2))
				for b.Loop() {
					benchRunesSink = decodeStdlibCore(input.data, out[:0])
				}
			})
			b.Run("simd_core", func(b *testing.B) {
				out := make([]rune, len(input.data))
				b.SetBytes(int64(len(input.data) * 2))
				for b.Loop() {
					benchRunesSink = decodeSIMD(input.data, out)
				}
			})
		})
	}
}

// decodeStdlibCore is the Go 1.27rc1 unicode/utf16.decode loop with its
// caller-owned buffer exposed for a no-allocation benchmark comparison.
func decodeStdlibCore(s []uint16, buf []rune) []rune {
	for i := 0; i < len(s); i++ {
		var ar rune
		switch r := s[i]; {
		case r < surrogateHighStart, surrogateEnd <= r:
			ar = rune(r)
		case surrogateHighStart <= r && r < surrogateLowStart && i+1 < len(s) &&
			surrogateLowStart <= s[i+1] && s[i+1] < surrogateEnd:
			ar = stdutf16.DecodeRune(rune(r), rune(s[i+1]))
			i++
		default:
			ar = replacementRune
		}
		buf = append(buf, ar)
	}
	return buf
}

func BenchmarkEncode(b *testing.B) {
	for _, input := range utf16BenchmarkRuneInputs() {
		b.Run(input.name, func(b *testing.B) {
			// Every rune can expand to two UTF-16 code units. Allocate the
			// reusable buffer before timing; each benchmark iteration measures
			// length calculation and encoding only.
			out := make([]uint16, 2*len(input.data))

			b.Run("stdlib_core", func(b *testing.B) {
				b.SetBytes(int64(len(input.data) * 4))
				for b.Loop() {
					capacity := encodedLengthStdlib(input.data)
					benchCodeUnitsSink = encodeStdlibCore(input.data, out[:capacity])
				}
			})
			b.Run("simd_core", func(b *testing.B) {
				b.SetBytes(int64(len(input.data) * 4))
				for b.Loop() {
					plan := planEncodeSIMD(input.data)
					benchCodeUnitsSink = encodeSIMDWithPlan(input.data, out, plan)
				}
			})
		})
	}
}

// encodedLengthStdlib is the first loop of Go 1.27rc1 unicode/utf16.Encode.
func encodedLengthStdlib(s []rune) int {
	n := len(s)
	for _, r := range s {
		if r >= surrogateOffset {
			n++
		}
	}
	return n
}

// encodeStdlibCore is the encoding loop of Go 1.27rc1 unicode/utf16.Encode,
// exposed here to benchmark it with a caller-owned output buffer.
func encodeStdlibCore(s []rune, out []uint16) []uint16 {
	n := 0
	for _, r := range s {
		switch stdutf16.RuneLen(r) {
		case 1:
			out[n] = uint16(r)
			n++
		case 2:
			r1, r2 := stdutf16.EncodeRune(r)
			out[n] = uint16(r1)
			out[n+1] = uint16(r2)
			n += 2
		default:
			out[n] = uint16(replacementRune)
			n++
		}
	}
	return out[:n]
}

func utf16BenchmarkInputs() []struct {
	name string
	data []uint16
} {
	const (
		oneKiBCodeUnits   = 1024 / 2
		sixtyFourKiBUnits = 64 * 1024 / 2
	)

	return []struct {
		name string
		data []uint16
	}{
		{name: "empty", data: nil},
		{name: "ascii_1KiB", data: repeatCodeUnits(oneKiBCodeUnits, []uint16{'A', 'S', 'C', 'I', 'I', ' ', 't', 'e', 'x', 't', ' '})},
		{name: "ascii_64KiB", data: repeatCodeUnits(sixtyFourKiBUnits, []uint16{'A', 'S', 'C', 'I', 'I', ' ', 't', 'e', 'x', 't', ' '})},
		{name: "bmp_mixed_64KiB", data: repeatCodeUnits(sixtyFourKiBUnits, []uint16{'G', 'o', ' ', 0x0416, 0x0435, 0x043B, 0x0442, 0x043E, ' ', 0x4E16, 0x754C, ' '})},
		{name: "dense_surrogates_64KiB", data: repeatCodeUnits(sixtyFourKiBUnits, []uint16{0xD83D, 0xDE00, 0xD83D, 0xDE80})},
		{name: "sparse_surrogates_64KiB", data: repeatCodeUnits(sixtyFourKiBUnits, []uint16{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 0xD83D, 0xDE00, 'j', 'k', 'l', 'm', 'n', 'o'})},
		{name: "unpaired_surrogates_64KiB", data: repeatCodeUnits(sixtyFourKiBUnits, []uint16{'a', 'b', 'c', 'd', 'e', 'f', 'g', 0xD800, 'h'})},
	}
}

func utf16BenchmarkRuneInputs() []struct {
	name string
	data []rune
} {
	const sixtyFourKiBRunes = 64 * 1024 / 4

	return []struct {
		name string
		data []rune
	}{
		{name: "empty", data: nil},
		{name: "ascii_64KiB", data: repeatRunes(sixtyFourKiBRunes, []rune("ASCII text "))},
		{name: "bmp_mixed_64KiB", data: repeatRunes(sixtyFourKiBRunes, []rune{'G', 'o', ' ', 0x0416, 0x0435, 0x043B, 0x0442, 0x043E, ' ', 0x4E16, 0x754C, ' '})},
		{name: "sparse_non_bmp_64KiB", data: repeatRunes(sixtyFourKiBRunes, []rune{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 0x1F600, 'i', 'j', 'k', 'l'})},
		{name: "dense_non_bmp_64KiB", data: repeatRunes(sixtyFourKiBRunes, []rune{0x1F600, 0x1F680})},
		{name: "invalid_64KiB", data: repeatRunes(sixtyFourKiBRunes, []rune{'a', 0xD800, 0x110000, -1, 'b'})},
	}
}

func repeatCodeUnits(length int, pattern []uint16) []uint16 {
	data := make([]uint16, length)
	for i := range data {
		data[i] = pattern[i%len(pattern)]
	}
	return data
}

func repeatRunes(length int, pattern []rune) []rune {
	data := make([]rune, length)
	for i := range data {
		data[i] = pattern[i%len(pattern)]
	}
	return data
}
