//go:build goexperiment.simd && amd64

package utf8

import (
	"bytes"
	"math/rand"
	"simd/archsimd"
	"testing"
	stdutf8 "unicode/utf8"
)

func TestRuneCountAVX2MatchesStandardLibrary(t *testing.T) {
	if !archsimd.X86.AVX2() {
		t.Skip("AVX2 is not available")
	}

	tests := [][]byte{
		nil,
		bytes.Repeat([]byte{'a'}, 1536),
		bytes.Repeat([]byte("hello, 世界 😀 "), 4096),
		bytes.Repeat([]byte("世界😀"), 4096),
		append(bytes.Repeat([]byte("世界😀"), 64), bytes.Repeat([]byte{'a'}, 1024)...),
		append(bytes.Repeat([]byte{'a'}, 509), []byte("😀")...),
		append(bytes.Repeat([]byte{'a'}, 511), 0x80),
		append(bytes.Repeat([]byte{'a'}, 511), 0xc0, 0xaf),
		append(bytes.Repeat([]byte{'a'}, 511), 0xed, 0xa0, 0x80),
		append(bytes.Repeat([]byte{'a'}, 1022), 0xe2, 0x82),
	}
	for _, data := range tests {
		assertRuneCountAVX2MatchesStandardLibrary(t, data)
	}

	rng := rand.New(rand.NewSource(2))
	for size := 0; size <= 2048; size++ {
		data := make([]byte, size)
		for range 4 {
			_, _ = rng.Read(data)
			assertRuneCountAVX2MatchesStandardLibrary(t, data)
		}
	}
}

func assertRuneCountAVX2MatchesStandardLibrary(t *testing.T, data []byte) {
	t.Helper()
	got, ok := runeCountAVX2(data)
	wantValid := stdutf8.Valid(data)
	if ok != wantValid {
		t.Fatalf("runeCountAVX2 input length %d validity = %v, want %v", len(data), ok, wantValid)
	}
	if ok {
		if want := stdutf8.RuneCount(data); got != want {
			t.Fatalf("runeCountAVX2 input length %d = %d, want %d", len(data), got, want)
		}
	}
}
