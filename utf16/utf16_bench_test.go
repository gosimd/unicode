package utf16_test

import (
	"testing"
	stdutf16 "unicode/utf16"

	simdutf16 "github.com/gosimd/unicode/utf16"
)

var benchRunesSink []rune

func BenchmarkDecode(b *testing.B) {
	for _, input := range utf16BenchmarkInputs() {
		b.Run(input.name, func(b *testing.B) {
			b.Run("stdlib", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data) * 2))
				for b.Loop() {
					benchRunesSink = stdutf16.Decode(input.data)
				}
			})
			b.Run("simd", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data) * 2))
				for b.Loop() {
					benchRunesSink = simdutf16.Decode(input.data)
				}
			})
		})
	}
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
		{name: "sparse_surrogates_64KiB", data: repeatCodeUnits(sixtyFourKiBUnits, []uint16{'a', 'b', 'c', 'd', 'e', 'f', 'g', 0xD83D, 0xDE00})},
		{name: "unpaired_surrogates_64KiB", data: repeatCodeUnits(sixtyFourKiBUnits, []uint16{'a', 'b', 'c', 'd', 'e', 'f', 'g', 0xD800, 'h'})},
	}
}

func repeatCodeUnits(length int, pattern []uint16) []uint16 {
	data := make([]uint16, length)
	for i := range data {
		data[i] = pattern[i%len(pattern)]
	}
	return data
}
