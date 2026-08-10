package utf8_test

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
	stdutf8 "unicode/utf8"

	simdutf8 "github.com/gosimd/unicode/simd/unicode/utf8"
)

var (
	benchBoolSink bool
	benchIntSink  int
)

func BenchmarkValid(b *testing.B) {
	benchmarkValid(b, utf8BenchByteInputs())
}

// BenchmarkValidSIMDUTF8Table supplies the simdutf8-style input matrix used by
// the generated two-column HTML report.
func BenchmarkValidSIMDUTF8Table(b *testing.B) {
	benchmarkValid(b, utf8SIMDUTF8BenchByteInputs())
}

func benchmarkValid(b *testing.B, inputs []struct {
	name string
	data []byte
}) {
	for _, input := range inputs {
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

// utf8SIMDUTF8BenchByteInputs mirrors simdutf8's benchmark matrix: a small
// sample of Latin, Cyrillic, Chinese, and emoji text at each target size. A
// multibyte sample can be up to three bytes longer than its target so that the
// input always ends at a UTF-8 boundary, just as in simdutf8.
func utf8SIMDUTF8BenchByteInputs() []struct {
	name string
	data []byte
} {
	const (
		latin    = "Lorem ipsum dolor sit amet, consectetur adipiscing elit. "
		cyrillic = "Съешь ещё этих мягких французских булок. "
		chinese  = "快速的棕色狐狸跳过懒狗。"
		emoji    = "😀👩🏽‍💻🚀✨❤️ "
	)

	charsets := []struct {
		name string
		data []byte
	}{
		{name: "1-latin", data: []byte(latin)},
		{name: "2-cyrillic", data: []byte(cyrillic)},
		{name: "3-chinese", data: []byte(chinese)},
		{name: "4-emoji", data: []byte(emoji)},
	}
	sizes := []int{2, 8, 64, 512, 4 * 1024, 64 * 1024, 128 * 1024}

	inputs := make([]struct {
		name string
		data []byte
	}, 0, len(charsets)*len(sizes)+2)
	inputs = append(inputs, struct {
		name string
		data []byte
	}{name: "0-empty/000000", data: nil})
	for _, charset := range charsets {
		for _, size := range sizes {
			data := validUTF8PrefixAtLeast(charset.data, size)
			inputs = append(inputs, struct {
				name string
				data []byte
			}{
				name: fmt.Sprintf("%s/%06d", charset.name, len(data)),
				data: data,
			})
		}
	}
	inputs = append(inputs, struct {
		name string
		data []byte
	}{name: "x-error/065536", data: append([]byte{0xff}, bytes.Repeat([]byte("a"), 65535)...)})
	return inputs
}

func validUTF8PrefixAtLeast(sample []byte, size int) []byte {
	data := make([]byte, 0, size+stdutf8.UTFMax-1)
	for len(data) < size {
		data = append(data, sample...)
	}
	for end := size; end <= len(data); end++ {
		if stdutf8.Valid(data[:end]) {
			return append([]byte(nil), data[:end]...)
		}
	}
	panic("benchmark sample has no valid UTF-8 prefix")
}

func TestSIMDUTF8BenchmarkInputs(t *testing.T) {
	inputs := utf8SIMDUTF8BenchByteInputs()
	if got, want := len(inputs), 30; got != want {
		t.Fatalf("benchmark input count = %d, want %d", got, want)
	}
	for _, input := range inputs {
		if input.name == "x-error/065536" {
			if stdutf8.Valid(input.data) {
				t.Fatal("error benchmark input is valid UTF-8")
			}
			continue
		}
		if !stdutf8.Valid(input.data) {
			t.Fatalf("%s is not valid UTF-8", input.name)
		}
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
