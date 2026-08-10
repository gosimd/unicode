package utf8

import stdutf8 "unicode/utf8"

// Compile-time API compatibility checks. Go has no package-level interface,
// so every exported function is checked against the standard-library type.
// This file is intentionally untagged: it verifies every supported build,
// including the SIMD and fallback implementations.
var (
	_ func([]byte, rune) []byte = stdutf8.AppendRune
	_ func([]byte) (rune, int)  = stdutf8.DecodeLastRune
	_ func(string) (rune, int)  = stdutf8.DecodeLastRuneInString
	_ func([]byte) (rune, int)  = stdutf8.DecodeRune
	_ func(string) (rune, int)  = stdutf8.DecodeRuneInString
	_ func([]byte, rune) int    = stdutf8.EncodeRune
	_ func([]byte) bool         = stdutf8.FullRune
	_ func(string) bool         = stdutf8.FullRuneInString
	_ func([]byte) int          = stdutf8.RuneCount
	_ func(string) int          = stdutf8.RuneCountInString
	_ func(rune) int            = stdutf8.RuneLen
	_ func(byte) bool           = stdutf8.RuneStart
	_ func([]byte) bool         = stdutf8.Valid
	_ func(rune) bool           = stdutf8.ValidRune
	_ func(string) bool         = stdutf8.ValidString

	_ func([]byte, rune) []byte = AppendRune
	_ func([]byte) (rune, int)  = DecodeLastRune
	_ func(string) (rune, int)  = DecodeLastRuneInString
	_ func([]byte) (rune, int)  = DecodeRune
	_ func(string) (rune, int)  = DecodeRuneInString
	_ func([]byte, rune) int    = EncodeRune
	_ func([]byte) bool         = FullRune
	_ func(string) bool         = FullRuneInString
	_ func([]byte) int          = RuneCount
	_ func(string) int          = RuneCountInString
	_ func(rune) int            = RuneLen
	_ func(byte) bool           = RuneStart
	_ func([]byte) bool         = Valid
	_ func(rune) bool           = ValidRune
	_ func(string) bool         = ValidString
)
