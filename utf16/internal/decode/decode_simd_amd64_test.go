//go:build goexperiment.simd && amd64

package decode

import (
	"fmt"
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

func TestDecodeAVX2SparsePairPositions(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("AVX2 is not available")
	}

	check := func(t *testing.T, input []uint16) {
		t.Helper()
		got := decodeAVX2(input, make([]rune, len(input)))
		if want := stdutf16.Decode(input); !reflect.DeepEqual(got, want) {
			t.Fatalf("decodeAVX2(%X) = %U, want %U", input, got, want)
		}
	}

	for pairLane := 0; pairLane < decodeSIMDChunkSize-1; pairLane++ {
		t.Run(fmt.Sprintf("lane_%d", pairLane), func(t *testing.T) {
			input := make([]uint16, 64)
			for i := range input {
				input[i] = 'a' + uint16(i%26)
			}
			input[pairLane] = surrogateHighStart + uint16(pairLane*31)
			input[pairLane+1] = surrogateLowStart + uint16(pairLane*47)
			check(t, input)
		})
	}

	t.Run("pair_across_chunk", func(t *testing.T) {
		input := make([]uint16, 64)
		for i := range input {
			input[i] = 'a' + uint16(i%26)
		}
		input[decodeSIMDChunkSize-1] = 0xDBFF
		input[decodeSIMDChunkSize] = 0xDFFF
		check(t, input)
	})

	for name, pair := range map[string][2]uint16{
		"two_high": {0xD800, 0xD801},
		"two_low":  {0xDC00, 0xDC01},
	} {
		t.Run(name, func(t *testing.T) {
			input := make([]uint16, 64)
			input[2], input[3] = pair[0], pair[1]
			check(t, input)
		})
	}
}

func TestDecodeAVX2SurrogateMaskBits(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("AVX2 is not available")
	}

	chunk := archsimd.LoadUint16x8Array(&[8]uint16{'a', 0xD800, 0xDC00, 'b', 'c', 'd', 'e', 'f'})
	mask := archsimd.BroadcastUint16x8(surrogateMask)
	marker := archsimd.BroadcastUint16x8(surrogateHighStart)
	zero := archsimd.BroadcastInt8x16(0)
	surrogates := chunk.And(mask).Equal(marker)
	got := ^surrogates.ToInt16x8().AsUint8x16().BitsToInt8().Equal(zero).ToBits()
	if got != 0x3C {
		t.Fatalf("surrogate mask = %#04x, want 0x003c", got)
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
