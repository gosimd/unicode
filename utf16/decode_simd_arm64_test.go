//go:build goexperiment.simd && arm64

package utf16

import (
	"reflect"
	"testing"
	stdutf16 "unicode/utf16"
)

func TestDecodeSIMDReusesOutput(t *testing.T) {
	input := []uint16{'a', 0xD83D, 0xDE00, 'b', 0xD800, 'c', 'd', 'e', 'f'}
	out := make([]rune, len(input))
	got := decodeSIMD(input, out)
	if want := stdutf16.Decode(input); !reflect.DeepEqual(got, want) {
		t.Fatalf("decodeSIMD(%X) = %U, want %U", input, got, want)
	}
	if len(got) > 0 && &got[0] != &out[0] {
		t.Fatal("decodeSIMD did not reuse the caller's output buffer")
	}
	if allocs := testing.AllocsPerRun(100, func() {
		benchRunesSink = decodeSIMD(input, out)
	}); allocs != 0 {
		t.Fatalf("decodeSIMD allocated %.0f times per run, want 0", allocs)
	}
}

func TestDecodeSIMDRejectsShortOutput(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("decodeSIMD accepted a short output buffer")
		}
	}()
	decodeSIMD([]uint16{'a'}, nil)
}

func TestDecodeSIMDUnrolledBlocks(t *testing.T) {
	clean := make([]uint16, 33)
	for i := range clean {
		clean[i] = uint16(0x4E00 + i)
	}

	pairAtBlockBoundary := make([]uint16, 40)
	for i := range pairAtBlockBoundary {
		pairAtBlockBoundary[i] = 'a'
	}
	pairAtBlockBoundary[31] = 0xD83D
	pairAtBlockBoundary[32] = 0xDE00

	for _, input := range [][]uint16{clean, pairAtBlockBoundary} {
		out := make([]rune, len(input))
		if got, want := decodeSIMD(input, out), stdutf16.Decode(input); !reflect.DeepEqual(got, want) {
			t.Fatalf("decodeSIMD(%X) = %U, want %U", input, got, want)
		}
	}
}
