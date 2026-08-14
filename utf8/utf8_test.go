package utf8_test

import (
	"bytes"
	"testing"
	stdutf8 "unicode/utf8"

	simdutf8 "github.com/gosimd/unicode/utf8"
)

func TestConstantsMatchStandardLibrary(t *testing.T) {
	if simdutf8.RuneError != stdutf8.RuneError {
		t.Fatalf("RuneError = %U, want %U", simdutf8.RuneError, stdutf8.RuneError)
	}
	if simdutf8.RuneSelf != stdutf8.RuneSelf {
		t.Fatalf("RuneSelf = %d, want %d", simdutf8.RuneSelf, stdutf8.RuneSelf)
	}
	if simdutf8.MaxRune != stdutf8.MaxRune {
		t.Fatalf("MaxRune = %U, want %U", simdutf8.MaxRune, stdutf8.MaxRune)
	}
	if simdutf8.UTFMax != stdutf8.UTFMax {
		t.Fatalf("UTFMax = %d, want %d", simdutf8.UTFMax, stdutf8.UTFMax)
	}
}

func TestAppendRuneMatchesStandardLibrary(t *testing.T) {
	tests := []struct {
		name   string
		prefix []byte
		r      rune
	}{
		{name: "ascii", prefix: nil, r: 'A'},
		{name: "two byte", prefix: []byte("prefix:"), r: '¢'},
		{name: "three byte", prefix: []byte("prefix:"), r: '世'},
		{name: "four byte", prefix: []byte("prefix:"), r: '😀'},
		{name: "rune error", prefix: []byte("prefix:"), r: stdutf8.RuneError},
		{name: "surrogate", prefix: []byte("prefix:"), r: 0xD800},
		{name: "above max rune", prefix: []byte("prefix:"), r: stdutf8.MaxRune + 1},
		{name: "negative", prefix: []byte("prefix:"), r: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := simdutf8.AppendRune(cloneBytes(tt.prefix), tt.r)
			want := stdutf8.AppendRune(cloneBytes(tt.prefix), tt.r)
			if !bytes.Equal(got, want) {
				t.Fatalf("AppendRune(%q, %U) = % x, want % x", tt.prefix, tt.r, got, want)
			}
		})
	}
}

func TestDecodeRuneMatchesStandardLibrary(t *testing.T) {
	for _, tt := range utf8ByteInputs() {
		t.Run(tt.name, func(t *testing.T) {
			gotRune, gotSize := simdutf8.DecodeRune(tt.data)
			wantRune, wantSize := stdutf8.DecodeRune(tt.data)
			if gotRune != wantRune || gotSize != wantSize {
				t.Fatalf("DecodeRune(% x) = (%U, %d), want (%U, %d)", tt.data, gotRune, gotSize, wantRune, wantSize)
			}
		})
	}
}

func TestDecodeRuneInStringMatchesStandardLibrary(t *testing.T) {
	for _, tt := range utf8StringInputs() {
		t.Run(tt.name, func(t *testing.T) {
			gotRune, gotSize := simdutf8.DecodeRuneInString(tt.data)
			wantRune, wantSize := stdutf8.DecodeRuneInString(tt.data)
			if gotRune != wantRune || gotSize != wantSize {
				t.Fatalf("DecodeRuneInString(%q) = (%U, %d), want (%U, %d)", tt.data, gotRune, gotSize, wantRune, wantSize)
			}
		})
	}
}

func TestDecodeLastRuneMatchesStandardLibrary(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "nil", data: nil},
		{name: "empty", data: []byte{}},
		{name: "ascii", data: []byte("A")},
		{name: "valid two byte", data: []byte("x¢")},
		{name: "valid three byte", data: []byte("x世")},
		{name: "valid four byte", data: []byte("x😀")},
		{name: "stray continuation at end", data: []byte{'x', 0x80}},
		{name: "truncated two byte at end", data: []byte{'x', 0xC2}},
		{name: "truncated three byte at end", data: []byte{'x', 0xE2, 0x82}},
		{name: "surrogate at end", data: []byte{'x', 0xED, 0xA0, 0x80}},
		{name: "above max at end", data: []byte{'x', 0xF4, 0x90, 0x80, 0x80}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRune, gotSize := simdutf8.DecodeLastRune(tt.data)
			wantRune, wantSize := stdutf8.DecodeLastRune(tt.data)
			if gotRune != wantRune || gotSize != wantSize {
				t.Fatalf("DecodeLastRune(% x) = (%U, %d), want (%U, %d)", tt.data, gotRune, gotSize, wantRune, wantSize)
			}
		})
	}
}

func TestDecodeLastRuneInStringMatchesStandardLibrary(t *testing.T) {
	tests := []struct {
		name string
		data string
	}{
		{name: "empty", data: ""},
		{name: "ascii", data: "A"},
		{name: "valid two byte", data: "x¢"},
		{name: "valid three byte", data: "x世"},
		{name: "valid four byte", data: "x😀"},
		{name: "stray continuation at end", data: string([]byte{'x', 0x80})},
		{name: "truncated two byte at end", data: string([]byte{'x', 0xC2})},
		{name: "truncated three byte at end", data: string([]byte{'x', 0xE2, 0x82})},
		{name: "surrogate at end", data: string([]byte{'x', 0xED, 0xA0, 0x80})},
		{name: "above max at end", data: string([]byte{'x', 0xF4, 0x90, 0x80, 0x80})},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRune, gotSize := simdutf8.DecodeLastRuneInString(tt.data)
			wantRune, wantSize := stdutf8.DecodeLastRuneInString(tt.data)
			if gotRune != wantRune || gotSize != wantSize {
				t.Fatalf("DecodeLastRuneInString(%q) = (%U, %d), want (%U, %d)", tt.data, gotRune, gotSize, wantRune, wantSize)
			}
		})
	}
}

func TestEncodeRuneMatchesStandardLibrary(t *testing.T) {
	tests := []struct {
		name string
		r    rune
	}{
		{name: "ascii", r: 'A'},
		{name: "boundary rune self minus one", r: stdutf8.RuneSelf - 1},
		{name: "boundary rune self", r: stdutf8.RuneSelf},
		{name: "two byte max", r: 0x07FF},
		{name: "three byte min", r: 0x0800},
		{name: "three byte", r: '世'},
		{name: "four byte", r: '😀'},
		{name: "max rune", r: stdutf8.MaxRune},
		{name: "surrogate min", r: 0xD800},
		{name: "surrogate max", r: 0xDFFF},
		{name: "above max rune", r: stdutf8.MaxRune + 1},
		{name: "negative", r: -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := []byte{0xAA, 0xAA, 0xAA, 0xAA}
			want := []byte{0xAA, 0xAA, 0xAA, 0xAA}
			gotN := simdutf8.EncodeRune(got, tt.r)
			wantN := stdutf8.EncodeRune(want, tt.r)
			if gotN != wantN || !bytes.Equal(got, want) {
				t.Fatalf("EncodeRune(%U) wrote (%d, % x), want (%d, % x)", tt.r, gotN, got, wantN, want)
			}
		})
	}
}

func TestFullRuneMatchesStandardLibrary(t *testing.T) {
	for _, tt := range utf8ByteInputs() {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := simdutf8.FullRune(tt.data), stdutf8.FullRune(tt.data); got != want {
				t.Fatalf("FullRune(% x) = %v, want %v", tt.data, got, want)
			}
		})
	}
}

func TestFullRuneInStringMatchesStandardLibrary(t *testing.T) {
	for _, tt := range utf8StringInputs() {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := simdutf8.FullRuneInString(tt.data), stdutf8.FullRuneInString(tt.data); got != want {
				t.Fatalf("FullRuneInString(%q) = %v, want %v", tt.data, got, want)
			}
		})
	}
}

func TestRuneCountMatchesStandardLibrary(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "nil", data: nil},
		{name: "empty", data: []byte{}},
		{name: "ascii", data: []byte("hello")},
		{name: "mixed", data: []byte("hello, 世界 😀")},
		{name: "stray continuation", data: []byte{'a', 0x80, 'b'}},
		{name: "truncated", data: []byte{'a', 0xE2, 0x82}},
		{name: "overlong", data: []byte{'a', 0xC0, 0xAF}},
		{name: "surrogate", data: []byte{'a', 0xED, 0xA0, 0x80}},
		{name: "above max", data: []byte{'a', 0xF4, 0x90, 0x80, 0x80}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := simdutf8.RuneCount(tt.data), stdutf8.RuneCount(tt.data); got != want {
				t.Fatalf("RuneCount(% x) = %d, want %d", tt.data, got, want)
			}
		})
	}
}

func TestRuneCountInStringMatchesStandardLibrary(t *testing.T) {
	for _, tt := range utf8StringInputsForWholeBuffer() {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := simdutf8.RuneCountInString(tt.data), stdutf8.RuneCountInString(tt.data); got != want {
				t.Fatalf("RuneCountInString(%q) = %d, want %d", tt.data, got, want)
			}
		})
	}
}

func TestRuneCountPrefixSweepMatchesStandardLibrary(t *testing.T) {
	suffixes := [][]byte{
		[]byte("x"),
		[]byte("¢"),
		[]byte("世"),
		[]byte("😀"),
		{0x80},
		{0xc0, 0xaf},
		{0xe0, 0x80, 0x80},
		{0xed, 0xa0, 0x80},
		{0xf4, 0x90, 0x80, 0x80},
		{0xe2, 0x82},
		{0xf0, 0x9f, 0x98},
	}

	for prefixLen := 0; prefixLen <= 96; prefixLen++ {
		prefix := bytes.Repeat([]byte{'a'}, prefixLen)
		for _, suffix := range suffixes {
			data := append(append([]byte(nil), prefix...), suffix...)
			if got, want := simdutf8.RuneCount(data), stdutf8.RuneCount(data); got != want {
				t.Fatalf("RuneCount(% x) = %d, want %d", data, got, want)
			}
		}
	}
}

func TestRuneCountWideBoundaryMatchesStandardLibrary(t *testing.T) {
	appendBytes := func(prefix []byte, suffix ...byte) []byte {
		return append(append([]byte(nil), prefix...), suffix...)
	}

	tests := []struct {
		name string
		data []byte
	}{
		{name: "ascii before window", data: bytes.Repeat([]byte{'a'}, 511)},
		{name: "ascii exact window", data: bytes.Repeat([]byte{'a'}, 512)},
		{name: "ascii after window", data: bytes.Repeat([]byte{'a'}, 513)},
		{name: "two byte across window", data: appendBytes(bytes.Repeat([]byte{'a'}, 511), []byte("¢")...)},
		{name: "three byte across window", data: appendBytes(bytes.Repeat([]byte{'a'}, 510), []byte("世")...)},
		{name: "four byte across window", data: appendBytes(bytes.Repeat([]byte{'a'}, 509), []byte("😀")...)},
		{name: "dirty then ascii", data: append(bytes.Repeat([]byte("世"), 171), bytes.Repeat([]byte{'a'}, 512)...)},
		{name: "ascii then dirty", data: append(bytes.Repeat([]byte{'a'}, 512), bytes.Repeat([]byte("世界😀"), 64)...)},
		{name: "truncated at window", data: appendBytes(bytes.Repeat([]byte{'a'}, 510), 0xe2, 0x82)},
		{name: "stray continuation after window", data: appendBytes(bytes.Repeat([]byte{'a'}, 512), 0x80)},
		{name: "overlong across window", data: appendBytes(bytes.Repeat([]byte{'a'}, 511), 0xc0, 0xaf)},
		{name: "invalid second byte across window", data: appendBytes(bytes.Repeat([]byte{'a'}, 511), 0xe0, 0x80, 0x80)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := simdutf8.RuneCount(tt.data), stdutf8.RuneCount(tt.data); got != want {
				t.Fatalf("RuneCount(% x) = %d, want %d", tt.data, got, want)
			}
			if got, want := simdutf8.RuneCountInString(string(tt.data)), stdutf8.RuneCountInString(string(tt.data)); got != want {
				t.Fatalf("RuneCountInString(% x) = %d, want %d", tt.data, got, want)
			}
		})
	}
}

func TestRuneLenMatchesStandardLibrary(t *testing.T) {
	tests := []rune{
		-1,
		0,
		'A',
		stdutf8.RuneSelf - 1,
		stdutf8.RuneSelf,
		0x07FF,
		0x0800,
		0xD7FF,
		0xD800,
		0xDFFF,
		0xE000,
		'世',
		'😀',
		stdutf8.MaxRune,
		stdutf8.MaxRune + 1,
	}

	for _, r := range tests {
		t.Run(stringForRuneName(r), func(t *testing.T) {
			if got, want := simdutf8.RuneLen(r), stdutf8.RuneLen(r); got != want {
				t.Fatalf("RuneLen(%U) = %d, want %d", r, got, want)
			}
		})
	}
}

func TestRuneStartMatchesStandardLibrary(t *testing.T) {
	for b := 0; b <= 0xFF; b++ {
		got := simdutf8.RuneStart(byte(b))
		want := stdutf8.RuneStart(byte(b))
		if got != want {
			t.Fatalf("RuneStart(0x%02x) = %v, want %v", b, got, want)
		}
	}
}

func TestValidMatchesStandardLibrary(t *testing.T) {
	for _, tt := range utf8StringInputsForWholeBuffer() {
		data := []byte(tt.data)
		t.Run(tt.name, func(t *testing.T) {
			if got, want := simdutf8.Valid(data), stdutf8.Valid(data); got != want {
				t.Fatalf("Valid(% x) = %v, want %v", data, got, want)
			}
		})
	}
}

func TestValidBoundaryMatchesStandardLibrary(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{name: "valid non ascii at 15", data: bytes.Join([][]byte{bytes.Repeat([]byte{'a'}, 15), []byte("世")}, nil)},
		{name: "valid non ascii at 16", data: bytes.Join([][]byte{bytes.Repeat([]byte{'a'}, 16), []byte("世")}, nil)},
		{name: "valid non ascii at 63", data: bytes.Join([][]byte{bytes.Repeat([]byte{'a'}, 63), []byte("😀")}, nil)},
		{name: "valid non ascii at 64", data: bytes.Join([][]byte{bytes.Repeat([]byte{'a'}, 64), []byte("😀")}, nil)},
		{name: "invalid at 15", data: bytes.Join([][]byte{bytes.Repeat([]byte{'a'}, 15), {0x80}}, nil)},
		{name: "invalid at 16", data: bytes.Join([][]byte{bytes.Repeat([]byte{'a'}, 16), {0x80}}, nil)},
		{name: "invalid at 17", data: bytes.Join([][]byte{bytes.Repeat([]byte{'a'}, 17), {0x80}}, nil)},
		{name: "invalid at 63", data: bytes.Join([][]byte{bytes.Repeat([]byte{'a'}, 63), {0x80}}, nil)},
		{name: "invalid at 64", data: bytes.Join([][]byte{bytes.Repeat([]byte{'a'}, 64), {0x80}}, nil)},
		{name: "invalid at 65", data: bytes.Join([][]byte{bytes.Repeat([]byte{'a'}, 65), {0x80}}, nil)},
		{name: "truncated after 64", data: bytes.Join([][]byte{bytes.Repeat([]byte{'a'}, 64), {0xE2, 0x82}}, nil)},
		{name: "overlong after 64", data: bytes.Join([][]byte{bytes.Repeat([]byte{'a'}, 64), {0xC0, 0xAF}}, nil)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := simdutf8.Valid(tt.data), stdutf8.Valid(tt.data); got != want {
				t.Fatalf("Valid(% x) = %v, want %v", tt.data, got, want)
			}
		})
	}
}

func TestValidWideBoundaryMatchesStandardLibrary(t *testing.T) {
	tests := []struct {
		name    string
		size    int
		offset  int
		payload []byte
	}{
		{name: "valid across 16-byte group", size: 1536, offset: 15, payload: []byte("😀")},
		{name: "valid across 64-byte vector", size: 1536, offset: 63, payload: []byte("😀")},
		{name: "valid across 512-byte window", size: 1536, offset: 511, payload: []byte("😀")},
		{name: "valid dirty then ascii window", size: 1536, offset: 520, payload: []byte("世")},
		{name: "stray continuation at window start", size: 1536, offset: 512, payload: []byte{0x80}},
		{name: "overlong across window", size: 1536, offset: 511, payload: []byte{0xc0, 0xaf}},
		{name: "surrogate across window", size: 1536, offset: 511, payload: []byte{0xed, 0xa0, 0x80}},
		{name: "above max across window", size: 1536, offset: 511, payload: []byte{0xf4, 0x90, 0x80, 0x80}},
		{name: "truncated at vector end", size: 1024, offset: 1022, payload: []byte{0xe2, 0x82}},
		{name: "invalid lead at vector end", size: 1024, offset: 1023, payload: []byte{0xff}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := bytes.Repeat([]byte{'a'}, tt.size)
			copy(data[tt.offset:], tt.payload)
			if got, want := simdutf8.Valid(data), stdutf8.Valid(data); got != want {
				t.Fatalf("Valid at offset %d = %v, want %v", tt.offset, got, want)
			}
			if got, want := simdutf8.ValidString(string(data)), stdutf8.ValidString(string(data)); got != want {
				t.Fatalf("ValidString at offset %d = %v, want %v", tt.offset, got, want)
			}
		})
	}
}

func TestValidInvalidLeadingByteAtSIMDChunkEnd(t *testing.T) {
	for _, prefixLen := range []int{15, 31, 47, 63, 79, 95} {
		for _, lead := range []byte{0xc0, 0xc1, 0xf5, 0xff} {
			data := append(bytes.Repeat([]byte{'a'}, prefixLen), lead)
			if got, want := simdutf8.Valid(data), stdutf8.Valid(data); got != want {
				t.Fatalf("Valid(% x) = %v, want %v", data, got, want)
			}
		}
	}
}

func TestValidPrefixSweepMatchesStandardLibrary(t *testing.T) {
	suffixes := []struct {
		name string
		data []byte
	}{
		{name: "ascii", data: []byte("x")},
		{name: "valid two byte", data: []byte("¢")},
		{name: "valid three byte min", data: []byte{0xe0, 0xa0, 0x80}},
		{name: "valid three byte before surrogate", data: []byte{0xed, 0x9f, 0xbf}},
		{name: "valid three byte after surrogate", data: []byte{0xee, 0x80, 0x80}},
		{name: "valid four byte min", data: []byte{0xf0, 0x90, 0x80, 0x80}},
		{name: "valid four byte max", data: []byte{0xf4, 0x8f, 0xbf, 0xbf}},
		{name: "stray continuation", data: []byte{0x80}},
		{name: "overlong two byte", data: []byte{0xc0, 0xaf}},
		{name: "overlong three byte", data: []byte{0xe0, 0x80, 0x80}},
		{name: "surrogate min", data: []byte{0xed, 0xa0, 0x80}},
		{name: "above max rune", data: []byte{0xf4, 0x90, 0x80, 0x80}},
		{name: "truncated two byte", data: []byte{0xc2}},
		{name: "truncated three byte", data: []byte{0xe2, 0x82}},
		{name: "truncated four byte", data: []byte{0xf0, 0x9f, 0x98}},
		{name: "invalid lead byte", data: []byte{0xff}},
	}

	for prefixLen := 0; prefixLen <= 96; prefixLen++ {
		prefix := bytes.Repeat([]byte{'a'}, prefixLen)
		for _, suffix := range suffixes {
			name := suffix.name + "_after_prefix_" + decimalString(prefixLen)
			t.Run(name, func(t *testing.T) {
				data := append(append([]byte(nil), prefix...), suffix.data...)
				if got, want := simdutf8.Valid(data), stdutf8.Valid(data); got != want {
					t.Fatalf("Valid(% x) = %v, want %v", data, got, want)
				}
			})
		}
	}
}

func TestValidRuneMatchesStandardLibrary(t *testing.T) {
	tests := []rune{
		-1,
		0,
		'A',
		stdutf8.RuneSelf - 1,
		stdutf8.RuneSelf,
		0x07FF,
		0x0800,
		0xD7FF,
		0xD800,
		0xDFFF,
		0xE000,
		'世',
		'😀',
		stdutf8.MaxRune,
		stdutf8.MaxRune + 1,
	}

	for _, r := range tests {
		t.Run(stringForRuneName(r), func(t *testing.T) {
			if got, want := simdutf8.ValidRune(r), stdutf8.ValidRune(r); got != want {
				t.Fatalf("ValidRune(%U) = %v, want %v", r, got, want)
			}
		})
	}
}

func TestValidStringMatchesStandardLibrary(t *testing.T) {
	for _, tt := range utf8StringInputsForWholeBuffer() {
		t.Run(tt.name, func(t *testing.T) {
			if got, want := simdutf8.ValidString(tt.data), stdutf8.ValidString(tt.data); got != want {
				t.Fatalf("ValidString(%q) = %v, want %v", tt.data, got, want)
			}
		})
	}
}

func FuzzValidMatchesStandardLibrary(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		bytes.Repeat([]byte{'a'}, 64),
		bytes.Repeat([]byte("hello, 世界 😀 "), 16),
		{0xc0},
		{0xe0, 0xa0, 0x80},
		{0xed, 0xa0, 0x80},
		{0xf4, 0x8f, 0xbf, 0xbf},
		{0xf4, 0x90, 0x80, 0x80},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if got, want := simdutf8.Valid(data), stdutf8.Valid(data); got != want {
			t.Fatalf("Valid(% x) = %v, want %v", data, got, want)
		}
	})
}

func FuzzRuneCountMatchesStandardLibrary(f *testing.F) {
	for _, seed := range [][]byte{
		nil,
		bytes.Repeat([]byte{'a'}, 64),
		bytes.Repeat([]byte("hello, 世界 😀 "), 16),
		{0x80},
		{0xc0, 0xaf},
		{0xe0, 0xa0, 0x80},
		{0xed, 0xa0, 0x80},
		{0xf4, 0x8f, 0xbf, 0xbf},
		{0xf4, 0x90, 0x80, 0x80},
	} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		if got, want := simdutf8.RuneCount(data), stdutf8.RuneCount(data); got != want {
			t.Fatalf("RuneCount(% x) = %d, want %d", data, got, want)
		}
	})
}

func cloneBytes(p []byte) []byte {
	if p == nil {
		return nil
	}
	return append([]byte(nil), p...)
}

func utf8ByteInputs() []struct {
	name string
	data []byte
} {
	return []struct {
		name string
		data []byte
	}{
		{name: "nil", data: nil},
		{name: "empty", data: []byte{}},
		{name: "ascii", data: []byte("A")},
		{name: "valid two byte", data: []byte("¢")},
		{name: "valid three byte", data: []byte("世")},
		{name: "valid four byte", data: []byte("😀")},
		{name: "truncated two byte", data: []byte{0xC2}},
		{name: "truncated three byte one byte", data: []byte{0xE2}},
		{name: "truncated three byte two bytes", data: []byte{0xE2, 0x82}},
		{name: "truncated four byte", data: []byte{0xF0, 0x9F, 0x98}},
		{name: "stray continuation", data: []byte{0x80}},
		{name: "overlong two byte", data: []byte{0xC0, 0xAF}},
		{name: "overlong three byte", data: []byte{0xE0, 0x80, 0xAF}},
		{name: "surrogate", data: []byte{0xED, 0xA0, 0x80}},
		{name: "above max rune", data: []byte{0xF4, 0x90, 0x80, 0x80}},
		{name: "invalid lead byte", data: []byte{0xFF}},
	}
}

func utf8StringInputs() []struct {
	name string
	data string
} {
	inputs := utf8ByteInputs()
	out := make([]struct {
		name string
		data string
	}, 0, len(inputs))
	for _, input := range inputs {
		out = append(out, struct {
			name string
			data string
		}{name: input.name, data: string(input.data)})
	}
	return out
}

func utf8StringInputsForWholeBuffer() []struct {
	name string
	data string
} {
	return []struct {
		name string
		data string
	}{
		{name: "empty", data: ""},
		{name: "ascii", data: "hello"},
		{name: "mixed", data: "hello, 世界 😀"},
		{name: "stray continuation middle", data: string([]byte{'a', 0x80, 'b'})},
		{name: "truncated at end", data: string([]byte{'a', 0xE2, 0x82})},
		{name: "overlong", data: string([]byte{'a', 0xC0, 0xAF})},
		{name: "surrogate", data: string([]byte{'a', 0xED, 0xA0, 0x80})},
		{name: "above max", data: string([]byte{'a', 0xF4, 0x90, 0x80, 0x80})},
	}
}

func stringForRuneName(r rune) string {
	return "U+" + string([]byte{
		"0123456789ABCDEF"[(uint32(r)>>20)&0xF],
		"0123456789ABCDEF"[(uint32(r)>>16)&0xF],
		"0123456789ABCDEF"[(uint32(r)>>12)&0xF],
		"0123456789ABCDEF"[(uint32(r)>>8)&0xF],
		"0123456789ABCDEF"[(uint32(r)>>4)&0xF],
		"0123456789ABCDEF"[uint32(r)&0xF],
	})
}

func decimalString(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
