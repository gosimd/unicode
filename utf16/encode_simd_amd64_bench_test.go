//go:build goexperiment.simd && amd64

package utf16

import (
	"simd/archsimd"
	"testing"
)

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
						capacity, mode := encodedLengthAVX2Profile(input.data)
						benchCodeUnitsSink = encodeAVX2(input.data, out, capacity, mode)
					}
				})
			}

			if archsimd.X86.AVX512() {
				b.Run("avx512_core", func(b *testing.B) {
					out := make([]uint16, 2*len(input.data))
					b.SetBytes(int64(len(input.data) * 4))
					for b.Loop() {
						capacity, mode := encodedLengthAVX512Profile(input.data)
						benchCodeUnitsSink = encodeAVX512(input.data, out, capacity, mode)
					}
				})
			}
		})
	}
}
