//go:build goexperiment.simd && amd64

package utf8

import (
	"simd/archsimd"
	stdutf8 "unicode/utf8"
	"unsafe"
)

// planEncodeAVX2 is the intentionally simple AVX2 baseline. Its optimization
// is left separate from the AVX-512 work.
func planEncodeAVX2(s []rune) encodePlan {
	size := len(s)
	allASCII := true
	allValid := true
	for _, r := range s {
		if r < 0 || r > stdutf8.MaxRune || r >= 0xd800 && r <= 0xdfff {
			r = stdutf8.RuneError
			allValid = false
		}
		switch {
		case r >= 0x10000:
			size += 3
		case r >= 0x800:
			size += 2
		case r >= 0x80:
			size++
		}
		allASCII = allASCII && r < 0x80
	}
	return encodePlan{size: size, allASCII: allASCII, allValid: allValid}
}

// encodeAVX2 is a correctness-first baseline with an eight-rune ASCII pack.
// Non-ASCII groups remain scalar until the dedicated AVX2 optimization pass.
func encodeAVX2(s []rune, out []byte, plan encodePlan) {
	if len(out) < plan.size+15 {
		panic("utf8: AVX2 encode output buffer too small")
	}
	inputBase := unsafe.Pointer(unsafe.SliceData(s))
	outputBase := unsafe.Pointer(unsafe.SliceData(out))
	threshold := archsimd.BroadcastUint32x8(0x80)
	i, n := 0, 0
	for ; len(s)-i >= 8; i += 8 {
		chunk := archsimd.LoadUint32x8Array(
			(*[8]uint32)(unsafe.Add(inputBase, uintptr(i)*4)),
		)
		if plan.allASCII || chunk.GreaterEqual(threshold).ToInt32x8().IsZero() {
			packed := chunk.GetLo().BitsToInt32().SaturateToUint16Concat(
				chunk.GetHi().BitsToInt32(),
			).SaturateToUint8()
			packed.StoreArray((*[16]uint8)(unsafe.Add(outputBase, uintptr(n))))
			n += 8
			continue
		}
		for j := i; j < i+8; j++ {
			n += stdutf8.EncodeRune(out[n:], s[j])
		}
	}
	for ; i < len(s); i++ {
		n += stdutf8.EncodeRune(out[n:], s[i])
	}
}
