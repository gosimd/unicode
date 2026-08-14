//go:build goexperiment.simd && arm64

package utf8

func runeCountSIMD(p []byte) (int, bool) {
	return runeCountSIMD128(p)
}
