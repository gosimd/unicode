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
		name     string
		input    []rune
		wantMode encodeAVX2Mode
	}{
		{name: "empty", input: nil, wantMode: encodeAVX2LowBMP},
		{name: "low_bmp_lane_order_and_tail", input: lowBMP, wantMode: encodeAVX2LowBMP},
		{name: "upper_bmp_lane_order_and_tail", input: upperBMP, wantMode: encodeAVX2Width16},
		{name: "sparse_non_bmp", input: []rune{'a', 'b', 0x1F600, 'c', 'd', 'e', 'f', 'g', 'h'}, wantMode: encodeAVX2MixedValid},
		{name: "dense_non_bmp", input: []rune{0x10000, 0x1F600, 0x10FFFF, 0x1F680}, wantMode: encodeAVX2AllNonBMP},
		{name: "invalid", input: []rune{-1, 0xD800, 0xDFFF, 0x110000, 'a'}, wantMode: encodeAVX2Scalar},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capacity, mode := encodedLengthAVX2Profile(tt.input)
			if mode != tt.wantMode {
				t.Fatalf("encodedLengthAVX2Profile mode = %d, want %d", mode, tt.wantMode)
			}
			got := encodeAVX2(tt.input, make([]uint16, 2*len(tt.input)), capacity, mode)
			if want := stdutf16.Encode(tt.input); !reflect.DeepEqual(got, want) {
				t.Fatalf("encodeAVX2(%U) = %X, want %X", tt.input, got, want)
			}
		})
	}
}

func TestEncodeAVX2AllNonBMPTails(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("AVX2 is not available")
	}

	for length := 0; length <= 65; length++ {
		input := make([]rune, length)
		for i := range input {
			input[i] = rune(surrogateOffset + (i*7919)%0x100000)
		}
		want := stdutf16.Encode(input)
		got := encodeAVX2AllNonBMPOnly(input, make([]uint16, len(want)), len(want))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("length %d: encodeAVX2AllNonBMPOnly = %X, want %X", length, got, want)
		}
	}
}

func TestEncodeAVX2MixedMasks(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("AVX2 is not available")
	}

	var sequence []rune
	for mask := 0; mask < 16; mask++ {
		input := []rune{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h'}
		for lane := 0; lane < 4; lane++ {
			if mask&(1<<lane) != 0 {
				input[lane] = rune(0x10000 + lane*0x12345)
			}
		}
		sequence = append(sequence, input[:4]...)
		want := stdutf16.Encode(input)
		got := encodeAVX2MixedValidOnly(input, make([]uint16, len(want)), len(want))
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("mask %#x: encodeAVX2MixedValidOnly = %X, want %X", mask, got, want)
		}
	}

	want := stdutf16.Encode(sequence)
	got := encodeAVX2MixedValidOnly(sequence, make([]uint16, len(want)), len(want))
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mask sequence: encodeAVX2MixedValidOnly = %X, want %X", got, want)
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
