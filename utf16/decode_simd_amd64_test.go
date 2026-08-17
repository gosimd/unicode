//go:build goexperiment.simd && amd64

package utf16

import (
	"reflect"
	"simd/archsimd"
	"testing"
	stdutf16 "unicode/utf16"
)

func TestDecodeAVX2MatchesStandardLibrary(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("AVX2 is not available")
	}

	densePairs := make([]uint16, 96)
	for i := 0; i < len(densePairs); i += 2 {
		pair := i / 2
		densePairs[i] = surrogateHighStart + uint16(pair%0x400)
		densePairs[i+1] = surrogateLowStart + uint16((pair*37)%0x400)
	}
	densePairs[len(densePairs)-2] = surrogateLowStart - 1
	densePairs[len(densePairs)-1] = surrogateEnd - 1
	interruptedPairs := append([]uint16(nil), densePairs[:32]...)
	interruptedPairs = append(interruptedPairs, 'A', 0xD800, 'B', 0xDC00)
	interruptedPairs = append(interruptedPairs, densePairs[32:]...)

	for _, input := range [][]uint16{
		nil,
		{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H'},
		{'A', 0xD800, 0xDC00, 'B', 0xD800, 'C', 0xDC00, 'D', 'E'},
		{0xD800, 0xDC00, 'A', 'B', 'C', 'D', 'E', 0xD800, 0xDC00},
		densePairs,
		interruptedPairs,
	} {
		got := decodeAVX2(input, make([]rune, len(input)))
		if want := stdutf16.Decode(input); !reflect.DeepEqual(got, want) {
			t.Fatalf("decodeAVX2(%X) = %U, want %U", input, got, want)
		}
	}
}

func FuzzDecodeAVX2MatchesStandardLibrary(f *testing.F) {
	if !archsimd.X86.AVX2() {
		f.Skip("AVX2 is not available")
	}

	for _, seed := range [][]byte{
		nil,
		{'A', 0, 'B', 0},
		{0x00, 0xD8, 0x00, 0xDC},
		{0x00, 0xD8, 0x00, 0xD8, 0x00, 0xDC},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		codeUnits := make([]uint16, len(data)/2)
		for i := range codeUnits {
			codeUnits[i] = uint16(data[2*i]) | uint16(data[2*i+1])<<8
		}
		got := decodeAVX2(codeUnits, make([]rune, len(codeUnits)))
		if want := stdutf16.Decode(codeUnits); !reflect.DeepEqual(got, want) {
			t.Fatalf("decodeAVX2(%X) = %U, want %U", codeUnits, got, want)
		}
	})
}
