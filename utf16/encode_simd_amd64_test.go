//go:build goexperiment.simd && amd64

package utf16

import (
	"reflect"
	"simd/archsimd"
	"testing"
	stdutf16 "unicode/utf16"
)

func TestEncodeAVX2ModesMatchStandardLibrary(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("AVX2 is not available")
	}

	lowBMP := make([]rune, 65)
	for i := range lowBMP {
		lowBMP[i] = rune((i*137 + 19) % surrogateHighStart)
	}
	upperBMP := make([]rune, 65)
	for i := range upperBMP {
		upperBMP[i] = rune(0xE000 + (i*251)%0x2000)
	}

	tests := []struct {
		name        string
		input       []rune
		wantLowMode bool
	}{
		{name: "empty", input: nil, wantLowMode: true},
		{name: "low_bmp_lane_order_and_tail", input: lowBMP, wantLowMode: true},
		{name: "upper_bmp_lane_order_and_tail", input: upperBMP},
		{name: "sparse_non_bmp", input: []rune{'a', 'b', 0x1F600, 'c', 'd', 'e', 'f', 'g', 'h'}},
		{name: "dense_non_bmp", input: []rune{0x10000, 0x1F600, 0x10FFFF, 0x1F680}},
		{name: "invalid", input: []rune{-1, 0xD800, 0xDFFF, 0x110000, 'a'}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capacity, mode := encodedLengthAVX2Profile(tt.input)
			if tt.wantLowMode && mode != encodeAVX2LowBMP {
				t.Fatalf("encodedLengthAVX2Profile mode = %d, want low-BMP mode", mode)
			}
			got := encodeAVX2(tt.input, make([]uint16, 2*len(tt.input)), capacity, mode)
			if want := stdutf16.Encode(tt.input); !reflect.DeepEqual(got, want) {
				t.Fatalf("encodeAVX2(%U) = %X, want %X", tt.input, got, want)
			}
		})
	}
}

func FuzzEncodeAVX2MatchesStandardLibrary(f *testing.F) {
	if !archsimd.X86.AVX2() {
		f.Skip("AVX2 is not available")
	}

	for _, seed := range [][]byte{
		nil,
		{0x41, 0, 0, 0, 0x00, 0x00, 0x01, 0},
		{0x00, 0xD8, 0, 0, 0x00, 0x00, 0x11, 0},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		runes := make([]rune, len(data)/4)
		for i := range runes {
			runes[i] = rune(uint32(data[4*i]) |
				uint32(data[4*i+1])<<8 |
				uint32(data[4*i+2])<<16 |
				uint32(data[4*i+3])<<24)
		}
		capacity, mode := encodedLengthAVX2Profile(runes)
		got := encodeAVX2(runes, make([]uint16, 2*len(runes)), capacity, mode)
		if want := stdutf16.Encode(runes); !reflect.DeepEqual(got, want) {
			t.Fatalf("encodeAVX2(%U) = %X, want %X", runes, got, want)
		}
	})
}
