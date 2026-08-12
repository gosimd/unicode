//go:build goexperiment.simd && amd64

package utf16

import (
	"simd/archsimd"
	"testing"
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
						benchRunesSink = decodeAVX2(input.data, out)
					}
				})
			}

			if archsimd.X86.AVX512() {
				b.Run("avx512_core", func(b *testing.B) {
					out := make([]rune, len(input.data))
					for b.Loop() {
						benchRunesSink = decodeAVX512(input.data, out)
					}
				})
			}
		})
	}
}
