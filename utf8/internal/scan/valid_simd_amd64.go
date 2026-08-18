//go:build goexperiment.simd && amd64

package scan

import "simd/archsimd"

// validSIMD dispatches after Valid has established that AVX2 is available.
func validSIMD(p []byte) bool {
	if archsimd.X86.AVX512() {
		return validAVX512(p)
	}
	return validAVX2(p)
}
