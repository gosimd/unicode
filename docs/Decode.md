# SIMD algorithm for `github.com/gosimd/unicode/utf8.Decode`

`Decode(s string)` has the same result as the Go language conversion
`[]rune(s)`. The SIMD implementation is selected on arm64 when the package is
built with `GOEXPERIMENT=simd`; all other builds use the language conversion.

## Public pipeline

The public operation has three stages:

1. `runeCountSIMD` validates the byte stream and counts runes in one traversal.
2. Invalid input falls back to `[]rune(s)`. Valid input allocates exactly the
   counted number of `rune` elements.
3. `decodeValidSIMD` decodes the already-validated input into that allocation.

The validation pass is intentional. It gives exact allocation and permits the
hot decoder to omit malformed-input branches. It also preserves Go's recovery
rule for malformed and truncated encodings without duplicating that state
machine in the SIMD decoder.

## ASCII windows

The outer loop loads four 16-byte vectors. A signed minimum proves that all 64
bytes are ASCII. Each vector is widened from eight-bit lanes to sixteen-bit
lanes and then to four 32-bit `rune` lanes, followed by sequential vector
stores. The loop therefore produces 64 runes without per-byte branches.

The final loop applies the same widening to one 16-byte ASCII vector at a time.

## Non-ASCII windows

For every byte, the signed predicate `byte < -64` identifies exactly the UTF-8
continuation range `0x80..0xBF`. Because `archsimd` does not expose a direct
ARM64 mask-to-bits operation, the implementation shifts true lanes to one,
multiplies the two eight-byte halves by powers of two, and reduces each half.
This creates a scalar continuation bitset.

Shifting the complement by one converts that bitset into end-of-sequence bits:
bit `i` is set when the byte at `i+1` starts another sequence. The low twelve
bits describe complete UTF-8 sequences beginning at the current input
position while the 16-byte load supplies four bytes of lookahead.

## Masked twelve-byte decoder

The twelve end bits index a 4096-entry dispatch table. Each entry contains:

- a selector for one of 256 shuffle rows;
- the number of input bytes consumed;
- whether the operation produces three or four runes.

The selected NEON `TBL` row right-aligns the bytes of up to four UTF-8
sequences in four `uint32` lanes. Fixed shifts and masks place the payload bits
at their Unicode scalar positions. Three-byte sequences use a per-lane
correction because their leading byte contributes a constant bit after the
generic composition.

Two masks bypass the general table path:

- `0xFFF` represents twelve ASCII bytes and widens them directly;
- `0xAAA` represents six dense two-byte sequences and decodes them as eight
  packed `uint16` lanes, storing only the six logical results.

The dense path's vector store has two spare lanes. If the exact public result
does not have room for those lanes at the final boundary, decoding switches to
the scalar tail instead of over-allocating.

The table layout follows the same twelve-byte/end-mask strategy used by
[simdutf's ARM64 UTF-8 to UTF-32 converter](https://github.com/simdutf/simdutf/blob/master/src/arm64/arm_convert_utf8_to_utf32.cpp), adapted to Go's `rune` type and the `simd/archsimd` API.

## Benchmark layers

`BenchmarkDecode` reports four separate operations:

- `stdlib_full`: `[]rune(s)`, including allocation;
- `simd_full`: public `Decode`, including validation, count, and allocation;
- `stdlib_core`: scalar `DecodeRuneInString` into caller-owned storage;
- `simd_core`: `decodeValidSIMD` into caller-owned storage, excluding
  validation, counting, and allocation.

The input matrix includes 1 KiB and 64 KiB ASCII, Latin/Cyrillic two-byte text,
CJK, emoji, mixed text, and early/late malformed UTF-8. Invalid inputs have no
`simd_core` case because the core contract requires validated input.
