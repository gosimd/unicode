package utf8_test

import (
	"bytes"
	"slices"
	"strings"
	"testing"
	stdutf8 "unicode/utf8"

	simdutf8 "github.com/gosimd/unicode/utf8"
)

var (
	benchBoolSink      bool
	benchByteSink      []byte
	benchIntSink       int
	benchRuneSliceSink []rune
	benchStringSink    string
)

func BenchmarkValid(b *testing.B) {
	benchmarkValid(b, utf8BenchByteInputs())
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

// BenchmarkEncode measures the full language-conversion APIs and equivalent
// caller-buffer core loops. The core variants exclude result allocation;
// simd_core also excludes the SIMD length-planning pass.
func BenchmarkEncode(b *testing.B) {
	for _, input := range utf8BenchStringInputs() {
		runes := []rune(input.data)
		encodedBytes := len(simdutf8.Encode(runes))
		simdPlan, simdEncodedBytes, simdAvailable := simdutf8.NewEncodeSIMDBenchmarkPlan(runes)
		b.Run(input.name, func(b *testing.B) {
			b.Run("stdlib_full", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(encodedBytes))
				for b.Loop() {
					benchStringSink = string(runes)
				}
			})
			b.Run("simd_full", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(encodedBytes))
				for b.Loop() {
					benchStringSink = simdutf8.Encode(runes)
				}
			})
			b.Run("stdlib_core", func(b *testing.B) {
				out := make([]byte, 0, encodedBytes)
				b.ReportAllocs()
				b.SetBytes(int64(encodedBytes))
				for b.Loop() {
					benchByteSink = encodeCore(runes, out[:0])
				}
			})
			b.Run("simd_core", func(b *testing.B) {
				if !simdAvailable {
					b.Skip("ARM64 SIMD encoder is unavailable")
				}
				out := make([]byte, simdEncodedBytes+15)
				b.ReportAllocs()
				b.SetBytes(int64(simdEncodedBytes))
				for b.Loop() {
					benchByteSink = simdutf8.EncodeSIMDCoreForBenchmark(runes, out, simdPlan)
				}
			})
		})
	}
}

// BenchmarkDecode keeps the public APIs separate from caller-buffer core loops.
// The core variants exclude validation, rune counting, and result allocation.
func BenchmarkDecode(b *testing.B) {
	for _, input := range utf8DecodeBenchStringInputs() {
		b.Run(input.name, func(b *testing.B) {
			b.Run("stdlib_full", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data)))
				for b.Loop() {
					benchRuneSliceSink = []rune(input.data)
				}
			})
			b.Run("simd_full", func(b *testing.B) {
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data)))
				for b.Loop() {
					benchRuneSliceSink = simdutf8.Decode(input.data)
				}
			})
			b.Run("stdlib_core", func(b *testing.B) {
				out := make([]rune, 0, simdutf8.RuneCountInString(input.data))
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data)))
				for b.Loop() {
					benchRuneSliceSink = decodeCore(input.data, out[:0])
				}
			})

			plan, decodedRunes, simdAvailable := simdutf8.NewDecodeSIMDBenchmarkPlan(input.data)
			b.Run("simd_core", func(b *testing.B) {
				if !simdAvailable {
					b.Skip("ARM64 SIMD decoder is unavailable for this input")
				}
				out := make([]rune, decodedRunes)
				b.ReportAllocs()
				b.SetBytes(int64(len(input.data)))
				for b.Loop() {
					benchRuneSliceSink = simdutf8.DecodeSIMDCoreForBenchmark(input.data, out, plan)
				}
			})
		})
	}
}

// encodeCore is the AppendRune form of string(s), with caller-owned output so
// BenchmarkEncode/core excludes result allocation.
func encodeCore(s []rune, out []byte) []byte {
	for _, r := range s {
		out = stdutf8.AppendRune(out, r)
	}
	return out
}

// decodeCore is the DecodeRuneInString form of []rune(s), with caller-owned
// output so BenchmarkDecode/core excludes result allocation.
func decodeCore(s string, out []rune) []rune {
	for len(s) > 0 {
		r, size := stdutf8.DecodeRuneInString(s)
		out = append(out, r)
		s = s[size:]
	}
	return out
}

func TestCoreConversionsMatchPublicAPI(t *testing.T) {
	for _, input := range utf8BenchStringInputs() {
		t.Run(input.name, func(t *testing.T) {
			runes := []rune(input.data)
			encoded := encodeCore(runes, make([]byte, 0, len(simdutf8.Encode(runes))))
			if got, want := string(encoded), simdutf8.Encode(runes); got != want {
				t.Fatalf("encodeCore = %q, want %q", got, want)
			}

			decoded := decodeCore(input.data, make([]rune, 0, simdutf8.RuneCountInString(input.data)))
			if want := simdutf8.Decode(input.data); !slices.Equal(decoded, want) {
				t.Fatalf("decodeCore = %U, want %U", decoded, want)
			}

			plan, decodedRunes, simdAvailable := simdutf8.NewDecodeSIMDBenchmarkPlan(input.data)
			if simdAvailable {
				simdDecoded := simdutf8.DecodeSIMDCoreForBenchmark(
					input.data,
					make([]rune, decodedRunes),
					plan,
				)
				if want := simdutf8.Decode(input.data); !slices.Equal(simdDecoded, want) {
					t.Fatalf("DecodeSIMDCoreForBenchmark = %U, want %U", simdDecoded, want)
				}
			}
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

func utf8DecodeBenchStringInputs() []struct {
	name string
	data string
} {
	return []struct {
		name string
		data string
	}{
		{name: "empty", data: ""},
		{name: "ascii_1KB", data: strings.Repeat("a", 1024)},
		{name: "ascii_64KB", data: strings.Repeat("a", 64*1024)},
		{name: "latin_64KB", data: strings.Repeat("¢", 32*1024)},
		{name: "cyrillic_64KB", data: strings.Repeat("Ж", 32*1024)},
		{name: "cjk_64KB", data: strings.Repeat("世", (64*1024)/3)},
		{name: "emoji_64KB", data: strings.Repeat("😀", 16*1024)},
		{name: "mixed_1KB", data: strings.Repeat("hello, 世界 ", 80)},
		{name: "mixed_64KB", data: strings.Repeat("hello, 世界 😀 ", 4096)},
		{name: "invalid_early", data: string(append([]byte{0x80}, bytes.Repeat([]byte{'a'}, 64*1024)...))},
		{name: "invalid_late", data: string(append(bytes.Repeat([]byte{'a'}, 64*1024), 0x80))},
	}
}

// BenchmarkReport supplies the stable, publication-oriented UTF-8 matrix used
// by cmd/benchreport. Keep it separate from the broader diagnostic benchmarks
// above so report changes do not silently remove regular benchmark coverage.
func BenchmarkReport(b *testing.B) {
	inputs := utf8ReportInputs()
	b.Run("utf8.Valid", func(b *testing.B) {
		for _, input := range inputs {
			b.Run(input.name, func(b *testing.B) {
				b.Run("gosimd", func(b *testing.B) {
					reportUTF8Metrics(b, input.data)
					for b.Loop() {
						benchBoolSink = simdutf8.Valid(input.data)
					}
					reportUTF8SizeMetrics(b, input.data)
				})
				b.Run("stdlib", func(b *testing.B) {
					reportUTF8Metrics(b, input.data)
					for b.Loop() {
						benchBoolSink = stdutf8.Valid(input.data)
					}
					reportUTF8SizeMetrics(b, input.data)
				})
			})
		}
	})
	b.Run("utf8.RuneCount", func(b *testing.B) {
		for _, input := range inputs {
			b.Run(input.name, func(b *testing.B) {
				b.Run("gosimd", func(b *testing.B) {
					reportUTF8Metrics(b, input.data)
					for b.Loop() {
						benchIntSink = simdutf8.RuneCount(input.data)
					}
					reportUTF8SizeMetrics(b, input.data)
				})
				b.Run("stdlib", func(b *testing.B) {
					reportUTF8Metrics(b, input.data)
					for b.Loop() {
						benchIntSink = stdutf8.RuneCount(input.data)
					}
					reportUTF8SizeMetrics(b, input.data)
				})
			})
		}
	})
}

func reportUTF8Metrics(b *testing.B, data []byte) {
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
}

func reportUTF8SizeMetrics(b *testing.B, data []byte) {
	b.ReportMetric(float64(len(data)), "input_bytes/op")
	b.ReportMetric(float64(stdutf8.RuneCount(data)), "chars/op")
}

// utf8ReportInputs uses a roughly 64 KiB input for every scenario. Multibyte
// inputs may be a few bytes larger so every buffer ends at a UTF-8 boundary.
func utf8ReportInputs() []struct {
	name string
	data []byte
} {
	const targetBytes = 64 * 1024
	patterns := []struct {
		name string
		text string
	}{
		{name: "ascii-only", text: "The quick brown fox jumps over the lazy dog. "},
		{name: "mixed", text: "Go: Hello, Привет, 世界! 😀 "},
		{name: "russian", text: "Съешь ещё этих мягких французских булок, да выпей чаю. "},
		{name: "chinese", text: "快速的棕色狐狸跳过懒狗。天地玄黄，宇宙洪荒。"},
	}

	inputs := make([]struct {
		name string
		data []byte
	}, 0, len(patterns))
	for _, pattern := range patterns {
		inputs = append(inputs, struct {
			name string
			data []byte
		}{name: pattern.name, data: repeatUTF8AtLeast(pattern.text, targetBytes)})
	}
	return inputs
}

func repeatUTF8AtLeast(sample string, size int) []byte {
	data := make([]byte, 0, size+len(sample))
	for len(data) < size {
		data = append(data, sample...)
	}
	return data
}

func TestUTF8ReportInputs(t *testing.T) {
	inputs := utf8ReportInputs()
	if got, want := len(inputs), 4; got != want {
		t.Fatalf("benchmark input count = %d, want %d", got, want)
	}
	for _, input := range inputs {
		if !stdutf8.Valid(input.data) {
			t.Fatalf("%s is not valid UTF-8", input.name)
		}
		if len(input.data) < 64*1024 {
			t.Fatalf("%s has %d bytes, want at least 64 KiB", input.name, len(input.data))
		}
	}
}
