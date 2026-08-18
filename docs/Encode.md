# SIMD algorithm for `github.com/gosimd/unicode/utf8.Encode`

`Encode(s []rune)` is exactly equivalent to the Go language conversion
`string(s)`. Negative runes, surrogate values, and values above `U+10FFFF` are
encoded as `RuneError`.

## Dispatch and planning

With `GOEXPERIMENT=simd`, arm64 selects NEON. amd64 requires AVX2 and selects
AVX-512 when the runtime reports the F, CD, BW, DQ, and VL feature bundle.
Machines without AVX2 use the language conversion.

The planner computes the exact byte length before allocation. AVX-512 first
checks 64-rune blocks for an all-ASCII input. General blocks normalize invalid
runes and count lanes crossing `0x80`, `0x800`, and `0x10000`. The plan records
whether all runes were valid, allowing the encoder to omit duplicate validity
checks. The output has fifteen private padding bytes because grouped compaction
uses full 16-byte stores; they are excluded from the returned string.

## AVX-512 encoder

The ASCII path processes four independent `Uint32x16` vectors and narrows each
with `VPMOVDB`, producing 64 bytes per iteration.

For non-ASCII input, sixteen runes are encoded in parallel. Dense two-byte
blocks narrow sixteen packed dwords with `VPMOVDW`; dense four-byte blocks are
stored directly. Three-byte and mixed blocks form one-, two-, three-, and
four-byte UTF-8 candidates in dword lanes.

Skylake lacks AVX-512 VBMI2, so byte compaction cannot use `VPCOMPRESSB`.
Instead, each ZMM candidate vector is divided into four 128-bit groups. Three
threshold masks form a four-rune selector for each group; a 4096-entry selector
table and a 256-row shuffle table select four independent `VPSHUFB` operations.
Overlapping 16-byte stores produce the contiguous output.

The final fewer-than-sixteen-rune tail uses `unicode/utf8.EncodeRune`.

## Benchmark layers

`BenchmarkEncode` keeps allocation and planning explicit:

- `stdlib_full` and `simd_full` include result allocation;
- `stdlib_core` writes scalar UTF-8 into caller-owned storage;
- `simd_core` uses a prepared plan and caller-owned padded output, reporting
  `0 B/op` and `0 allocs/op`.
