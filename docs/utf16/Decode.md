# SIMD algorithm for `github.com/gosimd/unicode/utf16.Decode`

`Decode(s []uint16)` has the same result as `unicode/utf16.Decode`.
Non-surrogate code units become runes directly, valid high/low surrogate pairs
become one non-BMP rune, and every unpaired surrogate becomes `U+FFFD`.

SIMD is an implementation detail. The decoded rune sequence does not depend
on the CPU architecture, runtime feature detection, or build flags.

## Dispatch and public pipeline

With `GOEXPERIMENT=simd`, arm64 uses NEON. On amd64, AVX2 is the minimum
runtime feature and AVX-512-capable hosts select a separate decoder. AMD64
hosts without AVX2, unsupported architectures, and builds without the SIMD
experiment call `unicode/utf16.Decode`.

The public SIMD operation allocates `len(s)` rune slots, the maximum possible
decoded length, and decodes into that caller-owned buffer. The returned slice
is shortened to the actual rune count after surrogate pairs have been
compacted.

Every SIMD implementation recognizes a surrogate code unit with the bit test:

```text
codeUnit & 0xF800 == 0xD800
```

Blocks without surrogate code units can be widened directly. Blocks with
surrogates use an architecture-specific pair path or the shared scalar state
machine.

## Scalar semantics and boundaries

The scalar loop preserves all `unicode/utf16.Decode` rules:

- a non-surrogate code unit becomes the same-valued rune;
- a high surrogate followed by a low surrogate becomes one non-BMP rune;
- an isolated high surrogate or any isolated low surrogate becomes `U+FFFD`.

The loop may consume one code unit beyond the nominal end of a rejected SIMD
block when its last lane is a high surrogate and the next code unit completes
the pair. This is necessary for exact behavior at vector boundaries; the next
SIMD iteration starts after the consumed low surrogate.

## ARM64 / NEON

The NEON loop works with eight-code-unit vectors. It first attempts an
unrolled 32-unit region by loading four independent vectors and combining
their surrogate predicates. If all four are clean, each vector is widened and
stored, giving the processor four independent load/widen/store chains.

The regular loop applies the same test to one eight-unit vector. Clean lanes
are zero-extended in two groups of four from `uint16` to 32-bit `rune` values.
The implementation uses fixed-size array loads and stores through an
internally checked caller buffer, avoiding per-chunk slice bounds checks.

If any lane is a surrogate, the complete eight-unit region is decoded by the
scalar state machine before SIMD probing resumes. The short final tail is also
scalar.

## AMD64 / AVX2 clean and sparse paths

The AVX2 outer loop first probes four independent eight-unit vectors. One mask
extraction determines whether all 32 code units are free of surrogates. Clean
vectors are widened to eight 32-bit lanes each and stored in source order.

When the 32-unit probe finds a surrogate, the decoder enters a sparse-run
helper. It processes eight-unit chunks and handles three cases:

- clean chunks are widened directly;
- one valid surrogate pair is decoded in place, and the following lanes are
  permuted left to remove the consumed low surrogate;
- malformed or more complicated chunks use the scalar state machine.

A valid pair beginning in lane seven may consume lane zero of the next chunk.
The decoder advances by nine input code units and produces eight runes, so it
does not decode the low surrogate twice.

After a scalar chunk produces exactly four runes from eight code units, the
decoder probes for dense valid surrogate pairs. Two 16-unit vectors are
validated against alternating high/low surrogate bases. `VPMADDWD`-style pair
dot products then combine every pair into one non-BMP rune, processing 32 code
units into 16 runes per iteration.

The sparse helper returns to the 32-unit outer probe after four consecutive
clean chunks. This avoids keeping surrogate-heavy control flow in the common
clean-BMP loop.

The AVX2 file uses XMM/YMM operations and `VPMOVMSKB`-style mask extraction;
it does not depend on AVX-512 mask registers.

## AMD64 / AVX-512

The AVX-512 decoder works with sixteen-code-unit vectors. Its unrolled path
loads and tests four vectors, covering 64 code units. One combined mask proves
that the region contains no surrogate code units; four 16-to-32-bit widening
stores then produce 64 runes.

The regular path widens one clean 16-unit vector. A region containing any
surrogate is decoded by the shared scalar state machine in eight-unit pieces.
Unlike the AVX2 decoder, the current AVX-512 implementation does not use the
sparse-pair or dense-pair SIMD helpers.

## Benchmark layers

The UTF-16 benchmark suite distinguishes the public API from the decoding
core:

- `utf16.Decode-full` compares public `Decode` with `unicode/utf16.Decode`,
  including their allocations;
- `utf16.Decode-core` compares both scalar and SIMD algorithms with
  caller-owned output, so both sides can report `0 B/op` and `0 allocs/op`;
- `BenchmarkDecodeAMD64Core` invokes AVX2 and AVX-512 directly on capable
  AMD64 hosts.

The focused matrix includes ASCII, mixed BMP, dense surrogate pairs, sparse
surrogate pairs, and unpaired surrogates. Clean-BMP throughput and
surrogate-heavy throughput are separate results: widening clean vectors does
not imply the same speedup when pair handling dominates.

## Correctness and source map

Public boundary tests and fuzzing compare `Decode` directly with
`unicode/utf16.Decode`. They cover pairs across SIMD boundaries, adjacent
malformed surrogates, isolated high and low surrogates, and arbitrary code
unit streams. AMD64 tests additionally exercise the direct AVX2 decoder,
every possible position of one valid pair, dense pairs, and surrogate mask
extraction.

The public wrapper is [`utf16.go`](../../utf16/utf16.go); the implementation
lives in `utf16/internal/decode`:

- [`decode_simd.go`](../../utf16/internal/decode/decode_simd.go): dispatch
  guard, constants, allocation, and shared scalar state machine;
- [`decode_simd_arm64.go`](../../utf16/internal/decode/decode_simd_arm64.go): NEON decoder;
- [`decode_simd_amd64.go`](../../utf16/internal/decode/decode_simd_amd64.go): runtime dispatch;
- [`decode_simd_avx2.go`](../../utf16/internal/decode/decode_simd_avx2.go): AVX2 clean,
  sparse-pair, and dense-pair paths;
- [`decode_simd_avx512.go`](../../utf16/internal/decode/decode_simd_avx512.go): AVX-512 clean
  widening path;
- [`decode_fallback.go`](../../utf16/internal/decode/decode_fallback.go): standard-library fallback.
