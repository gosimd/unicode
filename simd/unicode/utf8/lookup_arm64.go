//go:build goexperiment.simd && arm64

package utf8

import "simd/archsimd"

func lookupNibble(table archsimd.Uint8x16, indices archsimd.Uint8x16) archsimd.Uint8x16 {
	return table.LookupOrZero(indices)
}
