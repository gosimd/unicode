//go:build !goexperiment.simd || (!amd64 && !arm64)

package scan

import stdutf8 "unicode/utf8"

// RuneCount returns the number of runes in p. Erroneous and short encodings
// are treated as single runes of width 1 byte.
func RuneCount(p []byte) int {
	return stdutf8.RuneCount(p)
}

// RuneCountInString is like RuneCount but its input is a string.
func RuneCountInString(s string) int {
	return stdutf8.RuneCountInString(s)
}
