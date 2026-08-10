package main

import "testing"

func TestParseUsesMedianAndSelectsWinner(t *testing.T) {
	rows, err := parse([]byte(`BenchmarkValidSIMDUTF8Table/1-latin/000002/stdlib-10  1  12.00 ns/op  0 B/op  0 allocs/op
BenchmarkValidSIMDUTF8Table/1-latin/000002/simd-10    1  10.00 ns/op  0 B/op  0 allocs/op
BenchmarkValidSIMDUTF8Table/1-latin/000002/stdlib-10  1  14.00 ns/op  0 B/op  0 allocs/op
BenchmarkValidSIMDUTF8Table/1-latin/000002/simd-10    1  8.00 ns/op   0 B/op  0 allocs/op
BenchmarkValidSIMDUTF8Table/1-latin/000002/stdlib-10  1  13.00 ns/op  0 B/op  0 allocs/op
BenchmarkValidSIMDUTF8Table/1-latin/000002/simd-10    1  9.00 ns/op   0 B/op  0 allocs/op
`))
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(rows), 1; got != want {
		t.Fatalf("row count = %d, want %d", got, want)
	}
	if got, want := rows[0].Stdlib, 13.0; got != want {
		t.Fatalf("stdlib median = %v, want %v", got, want)
	}
	if got, want := rows[0].SIMD, 9.0; got != want {
		t.Fatalf("SIMD median = %v, want %v", got, want)
	}
	if !rows[0].SIMDWinner || rows[0].StdWinner {
		t.Fatal("SIMD should be the only winner")
	}
}
