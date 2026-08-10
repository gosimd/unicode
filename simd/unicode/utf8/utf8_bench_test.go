package utf8_test

import (
	"bytes"
	"strings"
	"testing"
	stdutf8 "unicode/utf8"

	simdutf8 "github.com/gosimd/utf/simd/unicode/utf8"
)

var (
	benchBoolSink bool
	benchIntSink  int
)

func BenchmarkValid(b *testing.B) {
	for _, input := range utf8BenchByteInputs() {
		b.Run(input.name, func(b *testing.B) {
			b.Run("stdlib", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data)))
				for b.Loop() {
					benchBoolSink = stdutf8.Valid(input.data)
				}
			})
			b.Run("simd", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data)))
				for b.Loop() {
					benchBoolSink = simdutf8.Valid(input.data)
				}
			})
		})
	}
}

func BenchmarkValidString(b *testing.B) {
	for _, input := range utf8BenchStringInputs() {
		b.Run(input.name, func(b *testing.B) {
			b.Run("stdlib", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data)))
				for b.Loop() {
					benchBoolSink = stdutf8.ValidString(input.data)
				}
			})
			b.Run("simd", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data)))
				for b.Loop() {
					benchBoolSink = simdutf8.ValidString(input.data)
				}
			})
		})
	}
}

func BenchmarkRuneCount(b *testing.B) {
	for _, input := range utf8BenchByteInputs() {
		b.Run(input.name, func(b *testing.B) {
			b.Run("stdlib", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data)))
				for b.Loop() {
					benchIntSink = stdutf8.RuneCount(input.data)
				}
			})
			b.Run("simd", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data)))
				for b.Loop() {
					benchIntSink = simdutf8.RuneCount(input.data)
				}
			})
		})
	}
}

func BenchmarkRuneCountInString(b *testing.B) {
	for _, input := range utf8BenchStringInputs() {
		b.Run(input.name, func(b *testing.B) {
			b.Run("stdlib", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data)))
				for b.Loop() {
					benchIntSink = stdutf8.RuneCountInString(input.data)
				}
			})
			b.Run("simd", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data)))
				for b.Loop() {
					benchIntSink = simdutf8.RuneCountInString(input.data)
				}
			})
		})
	}
}

func utf8BenchByteInputs() []struct {
	name string
	data []byte
} {
	ascii1KB := []byte(strings.Repeat("a", 1024))
	ascii64KB := []byte(strings.Repeat("a", 64*1024))
	mixed1KB := []byte(strings.Repeat("hello, 世界 ", 80))
	mixed64KB := bytes.Repeat([]byte("hello, 世界 😀 "), 4096)
	denseNonASCII := bytes.Repeat([]byte("世界😀"), 4096)
	invalidEarly := append([]byte{0x80}, ascii64KB...)
	invalidLate := append(append([]byte(nil), ascii64KB...), 0x80)
	truncatedLate := append(append([]byte(nil), ascii64KB...), 0xE2, 0x82)

	return []struct {
		name string
		data []byte
	}{
		{name: "empty", data: nil},
		{name: "ascii_1KB", data: ascii1KB},
		{name: "ascii_64KB", data: ascii64KB},
		{name: "mixed_1KB", data: mixed1KB},
		{name: "mixed_64KB", data: mixed64KB},
		{name: "dense_non_ascii", data: denseNonASCII},
		{name: "invalid_early", data: invalidEarly},
		{name: "invalid_late", data: invalidLate},
		{name: "truncated_late", data: truncatedLate},
	}
}

func utf8BenchStringInputs() []struct {
	name string
	data string
} {
	byteInputs := utf8BenchByteInputs()
	stringInputs := make([]struct {
		name string
		data string
	}, 0, len(byteInputs))
	for _, input := range byteInputs {
		stringInputs = append(stringInputs, struct {
			name string
			data string
		}{
			name: input.name,
			data: string(input.data),
		})
	}
	return stringInputs
}
