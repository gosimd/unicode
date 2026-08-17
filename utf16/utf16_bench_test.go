//go:build goexperiment.simd && (amd64 || arm64)

package utf16

import (
	"testing"
	stdutf16 "unicode/utf16"
)

var benchRunesSink []rune
var benchCodeUnitsSink []uint16

// BenchmarkReport supplies the stable, publication-oriented UTF-16 matrix
// consumed by cmd/benchreport. Full benchmarks call the public allocating API;
// core benchmarks reuse caller-owned output and include the length/planning
// pass while excluding output allocation.
func BenchmarkReport(b *testing.B) {
	inputs := utf16ReportInputs()
	b.Run("utf16.Encode-full", func(b *testing.B) {
		for _, input := range inputs {
			benchmarkReportEncodeFull(b, input.name, input.runes)
		}
	})
	b.Run("utf16.Encode-core", func(b *testing.B) {
		for _, input := range inputs {
			benchmarkReportEncodeCore(b, input.name, input.runes)
		}
	})
	b.Run("utf16.Decode-full", func(b *testing.B) {
		for _, input := range inputs {
			benchmarkReportDecodeFull(b, input.name, input.codeUnits)
		}
	})
	b.Run("utf16.Decode-core", func(b *testing.B) {
		for _, input := range inputs {
			benchmarkReportDecodeCore(b, input.name, input.codeUnits)
		}
	})
}

func benchmarkReportEncodeFull(b *testing.B, name string, data []rune) {
	b.Run(name, func(b *testing.B) {
		b.Run("gosimd", func(b *testing.B) {
			reportUTF16Metrics(b, len(data)*4, len(data))
			for b.Loop() {
				benchCodeUnitsSink = Encode(data)
			}
			reportUTF16SizeMetrics(b, len(data)*4, len(data))
		})
		b.Run("stdlib", func(b *testing.B) {
			reportUTF16Metrics(b, len(data)*4, len(data))
			for b.Loop() {
				benchCodeUnitsSink = stdutf16.Encode(data)
			}
			reportUTF16SizeMetrics(b, len(data)*4, len(data))
		})
	})
}

func benchmarkReportEncodeCore(b *testing.B, name string, data []rune) {
	b.Run(name, func(b *testing.B) {
		out := make([]uint16, 2*len(data))
		b.Run("gosimd", func(b *testing.B) {
			reportUTF16Metrics(b, len(data)*4, len(data))
			for b.Loop() {
				plan := planEncodeSIMD(data)
				benchCodeUnitsSink = encodeSIMDWithPlan(data, out, plan)
			}
			reportUTF16SizeMetrics(b, len(data)*4, len(data))
		})
		b.Run("stdlib", func(b *testing.B) {
			reportUTF16Metrics(b, len(data)*4, len(data))
			for b.Loop() {
				capacity := encodedLengthStdlib(data)
				benchCodeUnitsSink = encodeStdlibCore(data, out[:capacity])
			}
			reportUTF16SizeMetrics(b, len(data)*4, len(data))
		})
	})
}

func benchmarkReportDecodeFull(b *testing.B, name string, data []uint16) {
	chars := len(stdutf16.Decode(data))
	b.Run(name, func(b *testing.B) {
		b.Run("gosimd", func(b *testing.B) {
			reportUTF16Metrics(b, len(data)*2, chars)
			for b.Loop() {
				benchRunesSink = Decode(data)
			}
			reportUTF16SizeMetrics(b, len(data)*2, chars)
		})
		b.Run("stdlib", func(b *testing.B) {
			reportUTF16Metrics(b, len(data)*2, chars)
			for b.Loop() {
				benchRunesSink = stdutf16.Decode(data)
			}
			reportUTF16SizeMetrics(b, len(data)*2, chars)
		})
	})
}

func benchmarkReportDecodeCore(b *testing.B, name string, data []uint16) {
	chars := len(stdutf16.Decode(data))
	b.Run(name, func(b *testing.B) {
		out := make([]rune, len(data))
		b.Run("gosimd", func(b *testing.B) {
			reportUTF16Metrics(b, len(data)*2, chars)
			for b.Loop() {
				benchRunesSink = decodeSIMD(data, out)
			}
			reportUTF16SizeMetrics(b, len(data)*2, chars)
		})
		b.Run("stdlib", func(b *testing.B) {
			reportUTF16Metrics(b, len(data)*2, chars)
			for b.Loop() {
				benchRunesSink = decodeStdlibCore(data, out[:0])
			}
			reportUTF16SizeMetrics(b, len(data)*2, chars)
		})
	})
}

func reportUTF16Metrics(b *testing.B, inputBytes, chars int) {
	b.ReportAllocs()
	b.SetBytes(int64(inputBytes))
}

func reportUTF16SizeMetrics(b *testing.B, inputBytes, chars int) {
	b.ReportMetric(float64(inputBytes), "input_bytes/op")
	b.ReportMetric(float64(chars), "chars/op")
}

func utf16ReportInputs() []struct {
	name      string
	runes     []rune
	codeUnits []uint16
} {
	const (
		encodeRunes     = 64 * 1024 / 4
		decodeCodeUnits = 64 * 1024 / 2
	)
	patterns := []struct {
		name string
		data []rune
	}{
		{name: "ascii-only", data: []rune("The quick brown fox jumps over the lazy dog. ")},
		{name: "mixed", data: []rune("Go: Hello, Привет, 世界! 😀 ")},
		{name: "russian", data: []rune("Съешь ещё этих мягких французских булок, да выпей чаю. ")},
		{name: "chinese", data: []rune("快速的棕色狐狸跳过懒狗。天地玄黄，宇宙洪荒。")},
	}

	inputs := make([]struct {
		name      string
		runes     []rune
		codeUnits []uint16
	}, 0, len(patterns))
	for _, pattern := range patterns {
		inputs = append(inputs, struct {
			name      string
			runes     []rune
			codeUnits []uint16
		}{
			name:      pattern.name,
			runes:     repeatRunes(encodeRunes, pattern.data),
			codeUnits: repeatUTF16AtLeast(decodeCodeUnits, pattern.data),
		})
	}
	return inputs
}

func repeatUTF16AtLeast(length int, pattern []rune) []uint16 {
	encoded := stdutf16.Encode(pattern)
	data := make([]uint16, 0, length+len(encoded))
	for len(data) < length {
		data = append(data, encoded...)
	}
	return data
}

func TestUTF16ReportInputs(t *testing.T) {
	inputs := utf16ReportInputs()
	if got, want := len(inputs), 4; got != want {
		t.Fatalf("benchmark input count = %d, want %d", got, want)
	}
	for _, input := range inputs {
		if got, want := len(input.runes), 64*1024/4; got != want {
			t.Fatalf("%s Encode input has %d runes, want %d", input.name, got, want)
		}
		if len(input.codeUnits) < 64*1024/2 {
			t.Fatalf("%s Decode input has %d code units, want at least 32 Ki", input.name, len(input.codeUnits))
		}
		if reencoded := stdutf16.Encode(stdutf16.Decode(input.codeUnits)); !equalCodeUnits(reencoded, input.codeUnits) {
			t.Fatalf("%s Decode input is not well-formed UTF-16", input.name)
		}
	}
}

func equalCodeUnits(left, right []uint16) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

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
