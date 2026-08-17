//go:build !goexperiment.simd || (!amd64 && !arm64)

package main

func activeSIMD() string {
	return "disabled (build without supported GOEXPERIMENT=simd target)"
}
