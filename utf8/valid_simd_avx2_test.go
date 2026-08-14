//go:build goexperiment.simd && amd64

package utf8

import (
	"bytes"
	"math/rand"
	"simd/archsimd"
	"testing"
	stdutf8 "unicode/utf8"
)

func TestValidAVX2MatchesStandardLibrary(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("AVX2 is not available")
	}

	tests := []struct {
		name    string
		size    int
		offset  int
		payload []byte
	}{
		{name: "valid across first ymm", size: 1536, offset: 31, payload: []byte("😀")},
		{name: "valid across first window", size: 1536, offset: 509, payload: []byte("😀")},
		{name: "valid across second window", size: 1536, offset: 1022, payload: []byte("世")},
		{name: "dirty then ascii", size: 1536, offset: 100, payload: []byte("世")},
		{name: "ascii then dirty", size: 1536, offset: 700, payload: []byte("😀")},
		{name: "stray continuation at window", size: 1536, offset: 512, payload: []byte{0x80}},
		{name: "overlong across window", size: 1536, offset: 511, payload: []byte{0xc0, 0xaf}},
		{name: "surrogate across window", size: 1536, offset: 511, payload: []byte{0xed, 0xa0, 0x80}},
		{name: "above max across window", size: 1536, offset: 511, payload: []byte{0xf4, 0x90, 0x80, 0x80}},
		{name: "truncated at end", size: 1024, offset: 1022, payload: []byte{0xe2, 0x82}},
		{name: "invalid lead at end", size: 1024, offset: 1023, payload: []byte{0xff}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := bytes.Repeat([]byte{'a'}, tt.size)
			copy(data[tt.offset:], tt.payload)
			assertValidAVX2MatchesStandardLibrary(t, data)
		})
	}

	for _, data := range [][]byte{
		bytes.Repeat([]byte("hello, 世界 😀 "), 4096),
		bytes.Repeat([]byte("世界😀"), 4096),
		append(bytes.Repeat([]byte("世界😀"), 64), bytes.Repeat([]byte{'a'}, 1024)...),
	} {
		assertValidAVX2MatchesStandardLibrary(t, data)
	}

	rng := rand.New(rand.NewSource(1))
	for size := 0; size <= 2048; size++ {
		data := make([]byte, size)
		for range 4 {
			_, _ = rng.Read(data)
			assertValidAVX2MatchesStandardLibrary(t, data)
		}
	}
}

func assertValidAVX2MatchesStandardLibrary(t *testing.T, data []byte) {
	t.Helper()
	if got, want := validAVX2(data), stdutf8.Valid(data); got != want {
		t.Fatalf("validAVX2 input length %d = %v, want %v", len(data), got, want)
	}
}
