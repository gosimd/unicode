package utf16

import stdutf16 "unicode/utf16"

// Compile-time API compatibility checks. Go has no package-level interface,
// so every exported function is checked against the standard-library type.
// This file is intentionally untagged: it verifies every supported build,
// including future SIMD and fallback implementations.
var (
	_ func([]uint16, rune) []uint16 = stdutf16.AppendRune
	_ func([]uint16) []rune         = stdutf16.Decode
	_ func(rune, rune) rune         = stdutf16.DecodeRune
	_ func([]rune) []uint16         = stdutf16.Encode
	_ func(rune) (rune, rune)       = stdutf16.EncodeRune
	_ func(rune) bool               = stdutf16.IsSurrogate
	_ func(rune) int                = stdutf16.RuneLen

	_ func([]uint16, rune) []uint16 = AppendRune
	_ func([]uint16) []rune         = Decode
	_ func(rune, rune) rune         = DecodeRune
	_ func([]rune) []uint16         = Encode
	_ func(rune) (rune, rune)       = EncodeRune
	_ func(rune) bool               = IsSurrogate
	_ func(rune) int                = RuneLen
)
