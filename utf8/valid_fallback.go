//go:build !goexperiment.simd || (!amd64 && !arm64)

package utf8

import stdutf8 "unicode/utf8"

// Valid reports whether p consists entirely of valid UTF-8.
func Valid(p []byte) bool {
	return stdutf8.Valid(p)
}

// ValidString reports whether s consists entirely of valid UTF-8.
func ValidString(s string) bool {
	return stdutf8.ValidString(s)
}
