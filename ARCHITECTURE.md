# Architecture

## Design boundaries

The UTF-8 package, `github.com/gosimd/unicode/utf8`, is the stable user-facing
surface for UTF-8 operations. It mirrors `unicode/utf8` without exposing vector
types, masks, CPU checks, or architecture-specific behaviour. Its `Valid`,
`RuneCount`, `Encode`, and `Decode` methods select SIMD paths when they are
available.

The UTF-16 package, `github.com/gosimd/unicode/utf16`, mirrors
`unicode/utf16`. Its `Decode` function selects a whole-buffer SIMD path on
arm64 with NEON or amd64 with AVX2 when `GOEXPERIMENT=simd` is enabled.
`Encode` additionally selects a SIMD path for clean BMP blocks on arm64 and
AVX2-equipped amd64. Its other functions are standard-library facades.

## Validation dispatch

```text
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

`utf8.ValidString` and `RuneCountInString` create read-only, zero-copy byte
views when the SIMD implementation is available. Both supported and fallback
paths do not allocate.

## UTF-8 implementation layout

The public package is intentionally small. Operation-specific implementations
live below `utf8/internal`, while architecture suffixes and build tags keep the
NEON, AVX2, AVX-512, and fallback paths isolated inside each operation.

| Path | Responsibility |
| --- | --- |
| `utf8/whole_buffer.go` | Public `Valid`, `RuneCount`, `Encode`, and `Decode` wrappers. |
| `utf8/utf8.go`, `utf8/api_compat.go` | Standard-library-compatible API and compile-time compatibility checks. |
| `utf8/internal/scan` | Shared validation and rune-counting passes, including architecture-specific predicates, lookup tables, and fallbacks. |
| `utf8/internal/encode` | UTF-8 length planning, encoding, runtime dispatch, and fallback. |
| `utf8/internal/decode` | UTF-8 validation/count integration, decoding, runtime dispatch, and fallback. |

Within `scan`, `valid_simd_avx2.go` and `valid_simd_avx512.go` contain the
native wide validators, while the matching `rune_count_simd_*.go` files fuse
validation with continuation-byte counting. Its `aux_*.go` files collect the
small ISA-specific helpers for ASCII detection, nibble lookup, zero reduction,
and continuation counting. Within `encode` and `decode`, the `*_arm64.go`,
`*_avx2.go`, and `*_avx512.go` files contain the ISA-specific hot paths;
`*_amd64.go` files perform runtime dispatch. Every operation directory also
contains a build-tagged fallback.

The ARM64 loop and the amd64 baseline work on four 16-byte vectors (64 bytes).
The primary AVX2 `Valid` path tests 512-byte windows for ASCII and validates
dirty windows with sixteen native 32-byte vectors. `VPERM2I128` carries bytes
between each vector's two 128-bit shuffle groups. AVX-512 uses the same window
shape with eight native 64-byte vectors and a 64-bit lane permutation. Both
wide paths perform three grouped nibble lookups per vector and reduce
accumulated errors once per window. `RuneCount` uses the same native vector
widths and accumulates continuation-class flags while validating. A final
short tail is handled with a small scalar state machine.

See [docs/utf8/Valid.md](docs/utf8/Valid.md) for the detailed algorithm.

## SIMD UTF-8 encoding

`utf8.Encode` preserves the language conversion contract `string([]rune)`.
Negative runes, surrogate values, and values above `U+10FFFF` are normalized
to `RuneError`. On arm64 with `GOEXPERIMENT=simd`, a first NEON pass normalizes
those values for classification, computes the exact encoded length, and detects
an all-ASCII input. The result buffer is allocated once with 15 private padding
bytes, allowing each four-rune encoder to issue one full 16-byte store without
a partial-store branch. The returned string excludes the padding.

The ASCII path narrows sixteen `uint32` runes through `VXTN`, interleaves the
four packed vectors, and writes sixteen bytes. The general path constructs the
one-, two-, three-, and four-byte UTF-8 candidates in four `uint32` lanes. A
two-bit length code per lane selects one of 256 shuffle rows; NEON `TBL`
compacts the selected bytes into a contiguous result. Four-rune groups that mix
ASCII and wider encodings use the scalar encoder because constructing all four
SIMD candidates costs more for that shape.

On amd64, the common entry requires AVX2 and dynamically chooses AVX-512 when
available. The AVX2 branch is currently a correctness-first baseline with an
eight-rune ASCII pack; its non-ASCII path is deliberately left for a separate
optimization pass. The AVX-512 planner checks 64-rune ASCII blocks, otherwise
normalizes invalid runes and counts the three UTF-8 length thresholds in
16-rune vectors. It records whether the input is valid so the encoder does not
repeat normalization predicates.

AVX-512 ASCII encoding narrows four 16-rune vectors with `VPMOVDB`. Uniform
two-byte and four-byte blocks use `VPMOVDW` and direct dword stores. Three-byte
and mixed blocks construct UTF-8 candidates in sixteen dword lanes, then four
independent `VPSHUFB` operations compact each 128-bit group. A 4096-entry table
maps the three threshold masks to four-rune selectors. This uses only the
AVX-512F/CD/BW/DQ/VL bundle available on Skylake; it does not require
VBMI/VBMI2. See [docs/utf8/Encode.md](docs/utf8/Encode.md).

## SIMD UTF-8 decoding

`utf8.Decode` preserves the language conversion contract `[]rune(string)`.
The arm64, AVX2, and AVX-512 implementations first reuse their fused
validator/counter. Invalid input immediately delegates to the language
conversion, preserving its exact one-byte `RuneError` recovery semantics.
Valid input is allocated once at the exact rune count.

The decoder handles 64-byte ASCII windows by widening bytes to `uint16` and
then `uint32` before four-lane stores. For non-ASCII windows it converts the
continuation-byte predicate into a scalar end-of-sequence mask. The low twelve
bits address a 4096-entry dispatch table, which records how many source bytes
and output runes are covered and selects one of 256 NEON `TBL` shuffle rows.
The shuffle right-aligns up to four UTF-8 sequences in `uint32` lanes; masks,
shifts, and a three-byte correction then assemble Unicode scalar values. A
dense two-byte mask has its own six-rune widening path. The final short tail is
decoded scalar. See [docs/utf8/Decode.md](docs/utf8/Decode.md) for the detailed
dataflow.

On AVX2, `VPMOVMSKB` produces continuation bitsets for the same masked
twelve-byte strategy, with `VPSHUFB` replacing NEON `TBL`. Dense two-, three-,
and four-byte input is decoded in 64-, 48-, and 64-byte blocks respectively.

On AVX-512, an ASCII block widens 64 bytes through four `VPMOVZXBD`
conversions. A general block derives one 64-bit continuation mask, composes
UTF-32 candidates for each fixed 16-byte group with `VPALIGNR`, nibble-table
lookups, and variable shifts, then packs rune-start lanes with `VPCOMPRESSD`.
Dedicated 64-byte two- and four-byte paths and a 48-byte three-byte shuffle
path avoid compaction for uniform text. The common amd64 entry selects AVX-512
when available, otherwise AVX2; hosts without AVX2 use the language conversion.

## SIMD UTF-16 decoding

`utf16.Decode` processes eight `uint16` code units at a time. It identifies
surrogates with the equivalent bit test `codeUnit & 0xF800 == 0xD800`.
Architecture-specific loops prepare those two vector constants once before
processing chunks. Clean chunks are zero-extended into `rune` values: NEON
widens two groups of four lanes, while AVX2 widens all eight lanes. The AVX2
surrogate predicate uses `VPMOVMSKB` over an AVX comparison vector; it does
not use AVX-512 mask registers. AVX-512-capable AMD64 hosts select a separate
16-code-unit implementation which uses mask registers and a 512-bit
widen/store. AVX2 additionally recognizes one valid pair in an otherwise
clean chunk and dense runs of valid pairs. Other surrogate-containing regions
use the scalar state machine, so pairs at vector boundaries and malformed
sequences have the exact `unicode/utf16.Decode` result. Unsupported builds,
including amd64 without AVX2, delegate to the standard library.

The ARM64 NEON clean-chunk loop takes a caller-provided output buffer. It
checks its capacity once, then uses `unsafe.Add` with fixed-size array SIMD
loads and stores. This removes per-chunk slice bounds checks while preserving
the public `Decode` allocation and its standard-library-compatible result.
When at least 32 code units remain, it probes and widens four 8-unit NEON
chunks in one unrolled iteration; chunks containing a surrogate continue
through the existing 8-unit/scalar path.

See [docs/utf16/Decode.md](docs/utf16/Decode.md) for the detailed clean,
sparse-surrogate, dense-pair, and scalar dataflow.

## SIMD UTF-16 encoding

`utf16.Encode` first computes the exact standard-library allocation length:
every input rune at or above `U+10000` reserves a second code unit, even when
that rune is invalid. On arm64 with `GOEXPERIMENT=simd`, the encoder then
classifies two four-rune NEON vectors at a time. When all eight lanes are
valid BMP code points outside `U+D800`–`U+DFFF`, `VXTN` narrows both vectors;
their low 64-bit halves are interleaved and stored as eight contiguous
`uint16` values. On AVX2-equipped amd64, the length pass first reduces each
32-rune window with `VPMAXUD`; `VPSADBW` accumulates the non-BMP count while
packed 16-bit comparisons prove that the input contains valid Unicode scalar
values. Inputs entirely below `U+D800` select a predicate-free encoder
unrolled to 64 runes; `VPACKUSDW` narrows pairs of YMM vectors and `VPERMD`
restores sequential lane order. Inputs made entirely of non-BMP scalars use a
32-rune-unrolled loop that writes each high/low surrogate pair as one packed
`uint32`. Valid mixed input is encoded four runes at a time; a 16-entry
`VPSHUFB` table compacts the variable four-to-eight-code-unit result. Other
BMP input uses checked 16-rune blocks. Surrogate values and invalid runes use
the scalar fallback with exact `unicode/utf16.Encode` output and result
capacity. amd64 without AVX2 and all unsupported builds delegate `Encode` to
the standard library.

See [docs/utf16/Encode.md](docs/utf16/Encode.md) for the detailed planning,
NEON, AVX2, AVX-512, and scalar paths.

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
