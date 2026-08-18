//go:build goexperiment.simd && (amd64 || arm64)

package scan

import (
	"runtime"
	"simd/archsimd"
	stdutf8 "unicode/utf8"
	"unsafe"
)

// Valid reports whether p consists entirely of valid UTF-8.
func Valid(p []byte) bool {
	if runtime.GOARCH == "amd64" && !archsimd.X86.AVX2() {
		return stdutf8.Valid(p)
	}

	// This optimization avoids the need to recompute the capacity
	// when generating code for slicing p, bringing it to parity with
	// ValidString, which was 20% faster on long ASCII strings.
	p = p[:len(p):len(p)]
	return validSIMD(p)
}

// ValidString reports whether s consists entirely of valid UTF-8.
//
// It presents s to Valid without copying. Valid only reads its input, so the
// resulting byte slice does not violate string immutability.
func ValidString(s string) bool {
	if len(s) == 0 {
		return true
	}
	return Valid(unsafe.Slice(unsafe.StringData(s), len(s)))
}
