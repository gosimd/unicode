//go:build !goexperiment.simd || (!amd64 && !arm64)

package scan

import stdutf8 "unicode/utf8"

// CountValid validates p and counts its runes. The count is unspecified when
// valid is false.
func CountValid(p []byte) (count int, valid bool) {
	if !stdutf8.Valid(p) {
		return 0, false
	}
	return stdutf8.RuneCount(p), true
}
