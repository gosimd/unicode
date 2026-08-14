//go:build goexperiment.simd && amd64

package utf8

import (
	"bytes"
	"simd/archsimd"
	"strings"
	"testing"
	stdutf8 "unicode/utf8"
)

var validAMD64BenchmarkSink bool

func BenchmarkValidAMD64ASCII64KiB(b *testing.B) {
	data := []byte(strings.Repeat("a", 64*1024))

	benchmarkValidAMD64Path(b, "stdlib", data, stdutf8.Valid)
	if !archsimd.X86.AVX2() {
		return
	}
	benchmarkValidAMD64Path(b, "avx2_baseline", data, validAVX2Baseline)
	benchmarkValidAMD64Path(b, "avx2_ascii512", data, validAVX2)
	if archsimd.X86.AVX512() {
		benchmarkValidAMD64Path(b, "avx512", data, validAVX512)
	}
}

func BenchmarkValidAMD64NonASCII(b *testing.B) {
	inputs := []struct {
		name string
		data []byte
	}{
		{name: "mixed_64KB", data: bytes.Repeat([]byte("hello, 世界 😀 "), 4096)},
		{name: "dense_non_ascii", data: bytes.Repeat([]byte("世界😀"), 4096)},
		{name: "sparse_emoji_64KB", data: sparseEmoji64KiB()},
	}

	for _, input := range inputs {
		b.Run(input.name, func(b *testing.B) {
			benchmarkValidAMD64Path(b, "stdlib", input.data, stdutf8.Valid)
			if !archsimd.X86.AVX512() {
				return
			}
			benchmarkValidAMD64Path(b, "avx512", input.data, validAVX512)
		})
	}
}

func sparseEmoji64KiB() []byte {
	data := bytes.Repeat([]byte("a"), 64*1024)
	for offset := 4 * 1024; offset+len("😀") <= len(data); offset += 4 * 1024 {
		copy(data[offset:], "😀")
	}
	return data
}

func benchmarkValidAMD64Path(b *testing.B, name string, data []byte, valid func([]byte) bool) {
	b.Run(name, func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(data)))
		for b.Loop() {
			validAMD64BenchmarkSink = valid(data)
		}
	})
}
