//go:build goexperiment.simd && amd64

package utf8

import "simd/archsimd"

func continuationMask(chunk archsimd.Uint8x16) archsimd.Mask8x16 {
	return hasClassFlag(classFlags(chunk), utf8ClassContinuation)
}

func need1Mask(chunk archsimd.Uint8x16) archsimd.Mask8x16 {
	lead2, lead3, lead4 := leadMasks(chunk)
	return lead2.Or(lead3).Or(lead4)
}

func need2Mask(chunk archsimd.Uint8x16) archsimd.Mask8x16 {
	_, lead3, lead4 := leadMasks(chunk)
	return lead3.Or(lead4)
}

func need3Mask(chunk archsimd.Uint8x16) archsimd.Mask8x16 {
	_, _, lead4 := leadMasks(chunk)
	return lead4
}

func leadMasks(chunk archsimd.Uint8x16) (archsimd.Mask8x16, archsimd.Mask8x16, archsimd.Mask8x16) {
	flags := classFlags(chunk)
	low := lowNibbles(chunk)

	lead2 := hasClassFlag(flags, utf8ClassLead2).And(chunk.GreaterEqual(archsimd.BroadcastUint8x16(0xc2)))
	lead3 := hasClassFlag(flags, utf8ClassLead3)
	lead4 := hasClassFlag(flags, utf8ClassLead4).And(
		lookupNibble(archsimd.LoadUint8x16Array(&utf8ValidF0F4LowTable), low).NotEqual(archsimd.Uint8x16{}),
	)
	return lead2, lead3, lead4
}

func invalidLeadingBytes(chunk archsimd.Uint8x16) archsimd.Uint8x16 {
	high := highNibbles(chunk)
	low := lowNibbles(chunk)
	highC := maskBits(high.Equal(archsimd.BroadcastUint8x16(0x0c)))
	highF := maskBits(high.Equal(archsimd.BroadcastUint8x16(0x0f)))
	invalidC := lookupNibble(archsimd.LoadUint8x16Array(&utf8InvalidC0C1LowTable), low)
	invalidF := lookupNibble(archsimd.LoadUint8x16Array(&utf8InvalidF5FFLowTable), low)
	return highC.And(invalidC).Or(highF.And(invalidF))
}
