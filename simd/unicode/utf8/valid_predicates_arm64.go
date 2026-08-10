//go:build goexperiment.simd && arm64

package utf8

import "simd/archsimd"

// continuationMask reports bytes in the UTF-8 continuation range 0x80..0xbf.
// NEON maps unsigned range comparisons directly to VCMHS, so byte-range
// predicates are cheaper than classifying through a nibble lookup table.
func continuationMask(chunk archsimd.Uint8x16) archsimd.Mask8x16 {
	return chunk.And(archsimd.BroadcastUint8x16(0xc0)).
		Equal(archsimd.BroadcastUint8x16(0x80))
}

// need1Mask reports valid UTF-8 leading bytes that need at least one
// continuation byte: 0xc2..0xf4.
func need1Mask(chunk archsimd.Uint8x16) archsimd.Mask8x16 {
	return chunk.GreaterEqual(archsimd.BroadcastUint8x16(0xc2)).
		And(chunk.LessEqual(archsimd.BroadcastUint8x16(0xf4)))
}

// need2Mask reports valid UTF-8 leading bytes that need at least two
// continuation bytes: 0xe0..0xf4.
func need2Mask(chunk archsimd.Uint8x16) archsimd.Mask8x16 {
	return chunk.GreaterEqual(archsimd.BroadcastUint8x16(0xe0)).
		And(chunk.LessEqual(archsimd.BroadcastUint8x16(0xf4)))
}

// need3Mask reports valid UTF-8 leading bytes that need three continuation
// bytes: 0xf0..0xf4.
func need3Mask(chunk archsimd.Uint8x16) archsimd.Mask8x16 {
	return chunk.GreaterEqual(archsimd.BroadcastUint8x16(0xf0)).
		And(chunk.LessEqual(archsimd.BroadcastUint8x16(0xf4)))
}

// invalidLeadingBytes reports invalid UTF-8 leading bytes: C0, C1, and F5..FF.
// NEON can recognize both sets directly, avoiding nibble extraction and table
// lookups.
func invalidLeadingBytes(chunk archsimd.Uint8x16) archsimd.Uint8x16 {
	invalidC0C1 := chunk.And(archsimd.BroadcastUint8x16(0xfe)).
		Equal(archsimd.BroadcastUint8x16(0xc0))
	invalidF5FF := chunk.Greater(archsimd.BroadcastUint8x16(0xf4))
	return maskBits(invalidC0C1.Or(invalidF5FF))
}
