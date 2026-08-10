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
