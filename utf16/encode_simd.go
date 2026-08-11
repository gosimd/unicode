//go:build goexperiment.simd && (amd64 || arm64)

package utf16

func encodeScalar(s []rune, out []uint16, i, n, end int) (int, int) {
	for i < end {
		r := s[i]
		switch {
		case 0 <= r && r < surrogateHighStart, surrogateEnd <= r && r < surrogateOffset:
			out[n] = uint16(r)
			i++
			n++
		case surrogateOffset <= r && r <= 0x10FFFF:
			r -= surrogateOffset
			out[n] = uint16(surrogateHighStart + (r>>10)&0x3FF)
			out[n+1] = uint16(surrogateLowStart + r&0x3FF)
			i++
			n += 2
		default:
			out[n] = replacementRune
			i++
			n++
		}
	}
	return i, n
}
