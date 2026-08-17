# Architecture

## Design boundaries

The UTF-8 package, `github.com/gosimd/unicode/utf8`, is the stable user-facing
surface for UTF-8 operations. It mirrors `unicode/utf8` without exposing vector
types, masks, CPU checks, or architecture-specific behaviour. Its `Valid` and
`RuneCount` methods select the SIMD path when it is available.

The root package, `github.com/gosimd/unicode`, remains a compact convenience
facade for `Valid`, `ValidString`, and `RuneCount`. It delegates to the public
UTF-8 package.

The UTF-16 package, `github.com/gosimd/unicode/utf16`, mirrors
`unicode/utf16`. Its `Decode` function selects a whole-buffer SIMD path on
arm64 with NEON or amd64 with AVX2 when `GOEXPERIMENT=simd` is enabled.
`Encode` additionally selects a SIMD path for clean BMP blocks on arm64 and
AVX2-equipped amd64. Its other functions are standard-library facades.

## Validation dispatch

```text
utf.Valid([]byte) -> gosimd/unicode/utf8.Valid([]byte)
utf.RuneCount([]byte) -> gosimd/unicode/utf8.RuneCount([]byte)

gosimd/unicode/utf8.Valid([]byte)
gosimd/unicode/utf8.RuneCount([]byte)
        |
        +-- arm64 + GOEXPERIMENT=simd -> NEON implementation
        +-- amd64 + GOEXPERIMENT=simd + AVX-512 -> native 512-bit implementation
        +-- amd64 + GOEXPERIMENT=simd + AVX2 -> AVX2 implementations
        +-- otherwise -> matching unicode/utf8 function
```

The fallback is part of the correctness contract, not an exceptional path.
Every implementation must return the same result as its corresponding
`unicode/utf8` function.

`unicode/utf8.ValidString` and `RuneCountInString` create read-only,
zero-copy byte views when the SIMD implementation is available. The root
package exposes `ValidString`; both supported and fallback paths do not
allocate.

## SIMD implementation layout

| File group | Responsibility |
| --- | --- |
| `valid_simd.go` | SIMD `Valid` and `ValidString` entry points. |
| `rune_count_simd.go` | SIMD `RuneCount`, `RuneCountInString`, and shared 128-bit counting loop. |
| `rune_count_simd_avx2.go` | AVX2 lookup validator and fused continuation-byte sum. |
| `rune_count_simd_avx512.go` | AVX-512 lookup validator and fused continuation-byte sum. |
| `continuation_count_*.go` | Architecture-specific continuation-mask reduction for rune counting. |
| `utf8_simd_common.go` | Shared chunk sizes, byte classification, scalar tail state, and masks. |
| `valid_predicates_arm64.go` | NEON fused validation predicate and incomplete-sequence carry. |
| `ascii_arm64.go` | NEON ASCII detector using a signed minimum. |
| `valid_predicates_amd64.go` | 128-bit amd64 baseline predicate and incomplete-sequence carry. |
| `valid_simd_avx2.go` | Native 256-bit AVX2 validator, grouped carry, and wide ASCII fast path. |
| `valid_simd_avx512.go` | AVX-512 per-512-byte ASCII shortcut and lookup-based native wide validator. |
| `ascii_amd64.go` | amd64 ASCII detector. |
| `lookup_*`, `zero_*` | Architecture-specific table lookup and zero reduction helpers. |
| `valid_fallback.go`, `rune_count_fallback.go` | Non-SIMD standard-library fallbacks. |

The ARM64 loop and the amd64 baseline work on four 16-byte vectors (64 bytes).
The primary AVX2 `Valid` path tests 512-byte windows for ASCII and validates
dirty windows with sixteen native 32-byte vectors. `VPERM2I128` carries bytes
between each vector's two 128-bit shuffle groups. AVX-512 uses the same window
shape with eight native 64-byte vectors and a 64-bit lane permutation. Both
wide paths perform three grouped nibble lookups per vector and reduce
accumulated errors once per window. `RuneCount` uses the same native vector
widths and accumulates continuation-class flags while validating. A final
short tail is handled with a small scalar state machine.

See [docs/Valid.md](docs/Valid.md) for the detailed algorithm.

## SIMD UTF-16 decoding

`utf16.Decode` processes eight `uint16` code units at a time. It identifies
surrogates with the equivalent bit test `codeUnit & 0xF800 == 0xD800`.
Architecture-specific loops prepare those two vector constants once before
processing chunks. Clean chunks are zero-extended into `rune` values: NEON
widens two groups of four lanes, while AVX2 widens all eight lanes. The AVX2
surrogate predicate uses `VPMOVMSKB` over an AVX comparison vector; it does
not use AVX-512 mask registers. AVX-512-capable AMD64 hosts select a separate
16-code-unit implementation which uses mask registers and a 512-bit
widen/store. A chunk with any
surrogate is decoded by the scalar state machine one code unit at a time, so a
high surrogate at a vector boundary, valid pairs, and malformed sequences have the exact
`unicode/utf16.Decode` result. Unsupported builds, including amd64 without
AVX2, delegate to the standard library.

The ARM64 NEON clean-chunk loop takes a caller-provided output buffer. It
checks its capacity once, then uses `unsafe.Add` with fixed-size array SIMD
loads and stores. This removes per-chunk slice bounds checks while preserving
the public `Decode` allocation and its standard-library-compatible result.
When at least 32 code units remain, it probes and widens four 8-unit NEON
chunks in one unrolled iteration; chunks containing a surrogate continue
through the existing 8-unit/scalar path.

## SIMD UTF-16 encoding

`utf16.Encode` first computes the exact standard-library allocation length:
every input rune at or above `U+10000` reserves a second code unit, even when
that rune is invalid. On arm64 with `GOEXPERIMENT=simd`, the encoder then
classifies two four-rune NEON vectors at a time. When all eight lanes are
valid BMP code points outside `U+D800`–`U+DFFF`, `VXTN` narrows both vectors;
their low 64-bit halves are interleaved and stored as eight contiguous
`uint16` values. On AVX2-equipped amd64, the length pass first reduces each
32-rune window with `VPMAXUD`. Inputs entirely below `U+D800` select a
predicate-free encoder unrolled to 64 runes; `VPACKUSDW` narrows pairs of YMM
vectors and `VPERMD` restores sequential lane order. Other inputs use checked
16-rune blocks with a bitwise non-BMP/surrogate predicate and the same packing
sequence. Rejected blocks are encoded scalarly, covering non-BMP code points,
surrogate values, and invalid runes with exact `unicode/utf16.Encode` output
and result capacity. amd64 without AVX2 and all unsupported builds delegate
`Encode` to the standard library.

## SIMD rune counting

`RuneCount` validates and counts valid input in one traversal. NEON uses the
64-byte/16-byte shape and subtracts the population count of continuation
masks. AVX2 and AVX-512 use their native lookup validators on 512-byte windows.
Their current-byte lookup vectors contain the `0x02` class bit for every
continuation byte; `VPSADBW` accumulates those values in four AVX2 or eight
AVX-512 64-bit lanes, which are reduced once at the end. If validation fails, the
implementation falls back to `unicode/utf8.RuneCount`, preserving the
requirement that each malformed or truncated byte is a width-1 error rune.

## Correctness and performance policy

- `unicode/utf8` is the behavioural baseline.
- SIMD changes require regular tests, fuzzing against the standard library,
  and a fallback build check.
- Benchmark claims include input shape, architecture, toolchain, and
  `-benchmem`; they are observations, not API guarantees.
- Architecture-specific changes stay behind build tags and must preserve the
  same public result on every supported platform.
