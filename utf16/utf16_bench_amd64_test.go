//go:build goexperiment.simd && amd64

package utf16

import (
	"simd/archsimd"
	"testing"

	internaldecode "github.com/gosimd/unicode/utf16/internal/decode"
	internalencode "github.com/gosimd/unicode/utf16/internal/encode"
)

// BenchmarkDecodeAMD64Core measures the concrete AMD64 implementations
// independently of runtime dispatch and allocation cost.
func BenchmarkDecodeAMD64Core(b *testing.B) {
	for _, input := range utf16BenchmarkInputs() {
		b.Run(input.name, func(b *testing.B) {
			b.SetBytes(int64(len(input.data) * 2))

			if archsimd.X86.AVX2() {
				b.Run("avx2_core", func(b *testing.B) {
					out := make([]rune, len(input.data))
					for b.Loop() {
						benchRunesSink = internaldecode.DecodeAVX2CoreForBenchmark(input.data, out)
					}
				})
			}

			if archsimd.X86.AVX512() {
				b.Run("avx512_core", func(b *testing.B) {
					out := make([]rune, len(input.data))
					for b.Loop() {
						benchRunesSink = internaldecode.DecodeAVX512CoreForBenchmark(input.data, out)
					}
				})
			}
		})
	}
}

// BenchmarkEncodeAMD64Core measures the concrete AMD64 implementations
// independently of runtime dispatch and allocation cost.
func BenchmarkEncodeAMD64Core(b *testing.B) {
	for _, input := range utf16BenchmarkRuneInputs() {
		b.Run(input.name, func(b *testing.B) {
			if archsimd.X86.AVX2() {
				b.Run("avx2_core", func(b *testing.B) {
					out := make([]uint16, 2*len(input.data))
					b.SetBytes(int64(len(input.data) * 4))
					for b.Loop() {
						benchCodeUnitsSink = internalencode.EncodeAVX2CoreForBenchmark(input.data, out)
					}
				})
			}

			if archsimd.X86.AVX512() {
				b.Run("avx512_core", func(b *testing.B) {
					out := make([]uint16, 2*len(input.data))
					b.SetBytes(int64(len(input.data) * 4))
					for b.Loop() {
						benchCodeUnitsSink = internalencode.EncodeAVX512CoreForBenchmark(input.data, out)
					}
				})
			}
		})
	}
}
