# SIMD algorithm for `github.com/gosimd/unicode/utf8.Decode`

`Decode(s string)` has the same result as the Go language conversion
`[]rune(s)`. When the package is built with `GOEXPERIMENT=simd`, it selects a
NEON implementation on arm64 and dynamically selects AVX2 or AVX-512 on
amd64. Hosts without AVX2 and unsupported builds use the language conversion.

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

## ARM64 ASCII windows

The outer loop loads four 16-byte vectors. A signed minimum proves that all 64
bytes are ASCII. Each vector is widened from eight-bit lanes to sixteen-bit
lanes and then to four 32-bit `rune` lanes, followed by sequential vector
stores. The loop therefore produces 64 runes without per-byte branches.

The final loop applies the same widening to one 16-byte ASCII vector at a time.

## ARM64 non-ASCII windows

For every byte, the signed predicate `byte < -64` identifies exactly the UTF-8
continuation range `0x80..0xBF`. Because `archsimd` does not expose a direct
ARM64 mask-to-bits operation, the implementation shifts true lanes to one,
multiplies the two eight-byte halves by powers of two, and reduces each half.
This creates a scalar continuation bitset.

Shifting the complement by one converts that bitset into end-of-sequence bits:
bit `i` is set when the byte at `i+1` starts another sequence. The low twelve
bits describe complete UTF-8 sequences beginning at the current input
position while the 16-byte load supplies four bytes of lookahead.

## ARM64 masked twelve-byte decoder

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

## AVX2 decoder

The common amd64 entry checks AVX2 once, validates and counts the input, then
dispatches the prepared decode to AVX-512 when available or AVX2 otherwise.
The AVX2 ASCII loop loads four 16-byte vectors and widens them through
`VPMOVZXBW` and `VPMOVZXWD` into 64 UTF-32 lanes.

For mixed valid text, four `VPMOVMSKB` operations form a 64-bit continuation
bitset. Its complement drives the same twelve-byte table described above;
AVX2 `VPSHUFB` rearranges up to four sequences into `uint32` lanes before fixed
shifts, masks, and a correction produce runes. The final short tail is scalar.

Uniform non-ASCII text bypasses the general table. Dense two-byte and
four-byte input is decoded in 64-byte blocks with 256-bit word or dword
arithmetic. Dense three-byte input is decoded in 48-byte blocks by two
24-byte shuffles, each producing eight runes. The AVX2 function contains only
XMM/YMM operations and does not depend on AVX-512 mask registers.

## AVX-512 decoder

The amd64 decoder is selected only when runtime feature detection reports the
bundled AVX-512F, CD, BW, DQ, and VL feature set required by `archsimd`. A
64-byte signed comparison first detects an all-ASCII block. Four `VPMOVZXBD`
conversions widen its four 16-byte groups directly to four ZMM vectors of
UTF-32 lanes.

For a general valid block, one wide comparison produces a 64-bit continuation
mask. The decoder handles a fixed 16 input bytes per iteration and uses three
`VPALIGNR` operations to make the next three bytes available to every possible
rune start. Two high-nibble `VPSHUFB` tables select the leading-byte payload
mask and its initial shift of 0, 6, 12, or 18 bits. `VPMOVZXBD` and `VPSLLVD`
compose a UTF-32 candidate for every input byte. The complement of the
continuation mask selects actual rune starts; `VPCOMPRESSD` packs those lanes,
and a masked store writes exactly the number of produced runes.

Three common dense forms bypass that general work:

- 64 bytes of two-byte UTF-8 are reinterpreted as 32 pairs, decoded with
  16-bit shifts and masks, then widened with `VPMOVZXWD`;
- 48 bytes of three-byte UTF-8 are rearranged into 16 lead/middle/final byte
  vectors with nine `VPSHUFB` operations and decoded without compaction;
- 64 bytes of four-byte UTF-8 are decoded directly as 16 little-endian
  `uint32` lanes.

The fixed-width general loop may finish immediately before continuation bytes
that were already consumed as lookahead. It skips those bytes before entering
the scalar tail, avoiding duplicate output while keeping every vector load
within the input bounds.

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
