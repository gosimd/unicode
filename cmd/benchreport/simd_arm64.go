//go:build goexperiment.simd && arm64

package main

func activeSIMD() string {
	return "ARM NEON"
}
