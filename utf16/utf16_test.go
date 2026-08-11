package utf16

import (
	"reflect"
	"testing"
	stdutf16 "unicode/utf16"
)

func TestMatchesStandardLibrary(t *testing.T) {
	runes := []rune{-1, 0, 'A', 0xD7FF, 0xD800, 0xDC00, 0xDFFF, 0xE000, 0xFFFF, 0x10000, 0x10FFFF, 0x110000}
	for _, r := range runes {
		if got, want := IsSurrogate(r), stdutf16.IsSurrogate(r); got != want {
			t.Errorf("IsSurrogate(%U) = %v, want %v", r, got, want)
		}
		got1, got2 := EncodeRune(r)
		want1, want2 := stdutf16.EncodeRune(r)
		if got1 != want1 || got2 != want2 {
			t.Errorf("EncodeRune(%U) = %U, %U; want %U, %U", r, got1, got2, want1, want2)
		}
		if got, want := RuneLen(r), stdutf16.RuneLen(r); got != want {
			t.Errorf("RuneLen(%U) = %d, want %d", r, got, want)
		}
	}

	pairs := [][2]rune{{0xD800, 0xDC00}, {0xDBFF, 0xDFFF}, {0xD800, 0xD800}, {0xDC00, 0xD800}}
	for _, pair := range pairs {
		if got, want := DecodeRune(pair[0], pair[1]), stdutf16.DecodeRune(pair[0], pair[1]); got != want {
			t.Errorf("DecodeRune(%U, %U) = %U, want %U", pair[0], pair[1], got, want)
		}
	}

	input := []rune{'A', 0x10000, 0xD800, 0x110000}
	if got, want := Encode(input), stdutf16.Encode(input); !reflect.DeepEqual(got, want) {
		t.Errorf("Encode(%U) = %X, want %X", input, got, want)
	}

	codeUnits := []uint16{'A', 0xD800, 0xDC00, 0xD800, 0xDFFF}
	if got, want := Decode(codeUnits), stdutf16.Decode(codeUnits); !reflect.DeepEqual(got, want) {
		t.Errorf("Decode(%X) = %U, want %U", codeUnits, got, want)
	}

	if got, want := AppendRune([]uint16{'A'}, 0x10000), stdutf16.AppendRune([]uint16{'A'}, 0x10000); !reflect.DeepEqual(got, want) {
		t.Errorf("AppendRune() = %X, want %X", got, want)
	}
}

func TestEncodeMatchesStandardLibraryAtBoundaries(t *testing.T) {
	tests := []struct {
		name string
		data []rune
	}{
		{
			name: "clean_blocks_and_tail",
			data: []rune{'A', 0x07FF, 0xD7FF, 0xE000, 0xFFFF, 'Z', 0x4E16, 0x754C, 'x'},
		},
		{
			name: "non_bmp_at_block_boundary",
			data: []rune{'a', 'b', 'c', 'd', 'e', 'f', 'g', 0x1F600, 'h'},
		},
		{
			name: "surrogate_and_invalid_runes",
			data: []rune{'a', 0xD800, 0xDFFF, 0x110000, -1, 'b'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, want := Encode(tt.data), stdutf16.Encode(tt.data)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("Encode(%U) = %X, want %X", tt.data, got, want)
			}
			if cap(got) != cap(want) {
				t.Fatalf("cap(Encode(%U)) = %d, want %d", tt.data, cap(got), cap(want))
			}
		})
	}
}

func TestDecodeMatchesStandardLibraryAtBoundaries(t *testing.T) {
	tests := []struct {
		name string
		data []uint16
	}{
		{
			name: "clean_chunks_and_tail",
			data: []uint16{0, 'A', 0x07FF, 0xD7FF, 0xE000, 0xFFFF, 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L'},
		},
		{
			name: "pair_across_chunk_boundary",
			data: []uint16{'a', 'b', 'c', 'd', 'e', 'f', 'g', 0xD83D, 0xDE00, 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o'},
		},
		{
			name: "unpaired_high_at_last_chunk_lane",
			data: []uint16{'a', 'b', 'c', 'd', 'e', 'f', 'g', 0xD800, 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o'},
		},
		{
			name: "unpaired_low_at_first_chunk_lane",
			data: []uint16{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 0xDC00, 'i', 'j', 'k', 'l', 'm', 'n', 'o'},
		},
		{
			name: "adjacent_malformed_surrogates",
			data: []uint16{'a', 'b', 'c', 'd', 'e', 'f', 0xD800, 0xD801, 0xDC00, 0xDC01, 'g', 'h', 'i', 'j', 'k', 'l'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := Decode(tt.data), stdutf16.Decode(tt.data); !reflect.DeepEqual(got, want) {
				t.Fatalf("Decode(%X) = %U, want %U", tt.data, got, want)
			}
		})
	}
}

func FuzzDecodeMatchesStandardLibrary(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		{0x00, 0x00},
		{0x00, 0xD8, 0x00, 0xDC},
		{0x00, 0xD8},
		{0x00, 0xDC, 0x41, 0x00},
		{0xFF, 0xD7, 0x00, 0xE0},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		codeUnits := make([]uint16, len(data)/2)
		for i := range codeUnits {
			codeUnits[i] = uint16(data[2*i]) | uint16(data[2*i+1])<<8
		}
		if got, want := Decode(codeUnits), stdutf16.Decode(codeUnits); !reflect.DeepEqual(got, want) {
			t.Fatalf("Decode(%X) = %U, want %U", codeUnits, got, want)
		}
	})
}

func FuzzEncodeMatchesStandardLibrary(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		{0x00, 0x00, 0x00, 0x00},
		{0xFF, 0xD7, 0x00, 0x00, 0x00, 0xD8, 0x00, 0x00},
		{0xFF, 0xFF, 0x10, 0x00, 0x00, 0x00, 0x11, 0x00},
		{0xFF, 0xFF, 0xFF, 0xFF},
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
		got, want := Encode(runes), stdutf16.Encode(runes)
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("Encode(%U) = %X, want %X", runes, got, want)
		}
		if cap(got) != cap(want) {
			t.Fatalf("cap(Encode(%U)) = %d, want %d", runes, cap(got), cap(want))
		}
	})
}
