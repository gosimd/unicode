package utf

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestValid(t *testing.T) {
	tests := []struct {
		name string
		data []byte
		want bool
	}{
		{name: "empty", data: nil, want: true},
		{name: "ascii", data: []byte("hello"), want: true},
		{name: "multibyte", data: []byte("hello, 世界"), want: true},
		{name: "truncated", data: []byte{0xe2, 0x82}, want: false},
		{name: "stray continuation", data: []byte{0x80}, want: false},
		{name: "overlong", data: []byte{0xc0, 0xaf}, want: false},
		{name: "surrogate", data: []byte{0xed, 0xa0, 0x80}, want: false},
		{name: "too large", data: []byte{0xf4, 0x90, 0x80, 0x80}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Valid(tt.data); got != tt.want {
				t.Fatalf("Valid(%q) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

func TestValidMatchesStandardLibrary(t *testing.T) {
	inputs := [][]byte{
		nil,
		[]byte("ascii"),
		[]byte("hello, 世界"),
		{0xff},
		{0xe2, 0x82},
	}

	for _, input := range inputs {
		if got, want := Valid(input), utf8.Valid(input); got != want {
			t.Fatalf("Valid(%q) = %v, want %v", input, got, want)
		}
		if got, want := ValidString(string(input)), utf8.ValidString(string(input)); got != want {
			t.Fatalf("ValidString(%q) = %v, want %v", input, got, want)
		}
	}
}

func TestRuneCountMatchesStandardLibrary(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "empty", data: nil},
		{name: "ascii", data: []byte("hello")},
		{name: "mixed", data: []byte("hello, 世界 😀")},
		{name: "stray continuation", data: []byte{'a', 0x80, 'b'}},
		{name: "truncated", data: []byte{'a', 0xe2, 0x82}},
		{name: "overlong", data: []byte{'a', 0xc0, 0xaf}},
		{name: "surrogate", data: []byte{'a', 0xed, 0xa0, 0x80}},
		{name: "above max", data: []byte{'a', 0xf4, 0x90, 0x80, 0x80}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := RuneCount(tt.data), utf8.RuneCount(tt.data); got != want {
				t.Fatalf("RuneCount(% x) = %d, want %d", tt.data, got, want)
			}
		})
	}
}

func BenchmarkValidASCII1KB(b *testing.B) {
	data := []byte(strings.Repeat("a", 1024))
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for range b.N {
		if !Valid(data) {
			b.Fatal("unexpected invalid UTF-8")
		}
	}
}

func BenchmarkValidMixed1KB(b *testing.B) {
	data := []byte(strings.Repeat("hello, 世界 ", 80))
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))

	for range b.N {
		if !Valid(data) {
			b.Fatal("unexpected invalid UTF-8")
		}
	}
}
