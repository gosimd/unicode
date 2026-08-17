//go:build goexperiment.simd && amd64

package main

import "simd/archsimd"

func activeSIMD() string {
	if archsimd.X86.AVX512() {
		return "AVX-512"
	}
	if archsimd.X86.AVX2() {
		return "AVX2"
	}
	return "none (stdlib fallback)"
}
