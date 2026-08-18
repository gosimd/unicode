# SIMD algorithm for `github.com/gosimd/unicode/utf16.Encode`

`Encode(s []rune)` has the same result as `unicode/utf16.Encode`. Valid BMP
runes outside the surrogate range produce one code unit, valid non-BMP runes
produce a high/low surrogate pair, and invalid runes produce `U+FFFD`.

SIMD is an implementation detail. The returned code units and their order do
not depend on the CPU architecture, runtime feature detection, or build flags.

## Dispatch and allocation planning

With `GOEXPERIMENT=simd`, arm64 uses NEON. On amd64, AVX2 is the minimum
runtime feature; an AVX-512-capable host selects a separate implementation.
AMD64 hosts without AVX2, unsupported architectures, and builds without the
SIMD experiment call `unicode/utf16.Encode`.

The public operation first computes the output capacity, allocates once, and
then encodes into that allocation. The capacity rule deliberately matches the
standard library: it starts at `len(s)` and adds one for every rune greater
than or equal to `U+10000`. This includes out-of-range positive values even
though they are later replaced by one `U+FFFD` code unit. Consequently, the
returned slice length can be smaller than its capacity for invalid input.

On amd64, the length pass also records an internal `encodingPlan`. It reuses
the non-BMP density and validity information already discovered by the
mandatory pass to choose an encoding mode without allocating a bitmap or
running another whole-input classification pass.

## Scalar semantics

The shared scalar loop implements the complete contract:

- runes in `[U+0000, U+D7FF]` and `[U+E000, U+FFFF]` are narrowed to one
  `uint16`;
- runes in `[U+10000, U+10FFFF]` are converted to a high/low surrogate pair;
- negative runes, surrogate values, and values above `U+10FFFF` become
  `U+FFFD`.

SIMD loops hand rejected blocks to this scalar loop. They do not approximate
or reinterpret invalid values.

## ARM64 / NEON

The ARM64 length pass counts runes greater than or equal to `U+10000` four at
a time. The encoding loop examines two four-rune vectors per iteration. A
block is clean when every lane is a valid BMP code point outside
`U+D800`–`U+DFFF`.

For a clean eight-rune block, `VXTN` narrows both vectors from 32-bit runes to
16-bit code units. `VZIP1`-style interleaving joins the useful low halves into
one ordered eight-lane store. A rejected block, including non-BMP or invalid
input, is encoded by the scalar loop. The final short tail is scalar as well.

## AMD64 / AVX2 planning

The AVX2 planner first probes 32-rune windows for the common low-BMP case in
which every rune is below `U+D800`. If the whole input satisfies that
predicate, the encoder can omit value checks from its hot loop.

Otherwise, the planner processes 32 runes per iteration while it:

- counts lanes greater than or equal to `U+10000` for exact capacity;
- rejects surrogate values;
- rejects values above `U+10FFFF`;
- records whether every rune is a valid Unicode scalar value.

The resulting plan selects one of these shapes:

- `low BMP`: predicate-free 64-rune iterations;
- `all non-BMP`: 32 valid non-BMP runes per unrolled iteration;
- `mixed valid`: table-driven four-rune compaction;
- checked 16-rune or 8-rune BMP windows for sparse non-BMP input;
- scalar encoding when vector probing would cost more than it saves.

## AMD64 / AVX2 encoding

The checked BMP paths load either sixteen or eight runes. They reject lanes
with bits above the BMP or inside the surrogate range. Clean lanes are packed
with grouped saturating 32-to-16-bit narrowing, and `VPERMD` restores source
lane order after AVX2's lane-local packing.

The low-BMP path has already proved every rune is below `U+D800`, so it omits
those checks and processes 64 runes per iteration.

For all-non-BMP valid input, each 32-bit rune is converted to a packed
high/low surrogate pair. Four independent eight-rune vectors produce 32 pairs
per unrolled iteration.

For valid mixed input, four runes form four candidate packed pairs. The
non-BMP mask indexes a 16-row `VPSHUFB` table that removes the unused second
code unit of BMP lanes. The loop advances by four input runes and by four plus
the mask population count in the output.

All AVX2 entry points use only AVX2-width operations. AVX-512 narrowing is
kept in a separate file so AVX2-only machines cannot execute an accidental
AVX-512 instruction.

## AMD64 / AVX-512

The AVX-512 planner counts non-BMP runes sixteen at a time. The density found
during that pass selects checked windows of 64, 16, or 8 runes; dense input
selects the scalar loop. The eight-rune mode reuses the AVX2 short path.

The 16- and 64-rune paths use an AVX-512 mask to prove that every lane is a
valid BMP code point outside the surrogate range. Clean 16-rune vectors are
narrowed directly with the AVX-512 form emitted by `TruncToUint16`. A rejected
region is passed to the scalar encoder in eight-rune pieces, preserving exact
replacement and surrogate-pair behavior.

## Benchmark layers

The UTF-16 benchmark suite keeps allocation-inclusive and caller-buffer work
separate:

- `utf16.Encode-full` compares the public `Encode` operation with
  `unicode/utf16.Encode`, including allocation;
- `utf16.Encode-core` compares both algorithms with caller-owned output;
  planning and exact-length calculation remain inside the timed region;
- `BenchmarkEncodeAMD64Core` measures AVX2 and AVX-512 implementations
  directly on capable AMD64 hosts.

The input matrix includes ASCII, mixed BMP, sparse and dense non-BMP text, and
invalid runes. Results are shape-specific: a clean BMP gain must not be
presented as the expected result for surrogate-heavy or invalid input.

## Correctness and source map

Public boundary tests and fuzzing compare `Encode` directly with
`unicode/utf16.Encode`, including surrogate values, negative runes,
`U+10FFFF`, out-of-range values, and SIMD block boundaries. AMD64 tests also
exercise AVX2 mode selection, every four-rune mixed compaction mask, tails,
and the concrete AVX2 encoder.

The implementation is split across:

- [`encode_simd.go`](../../utf16/encode_simd.go): shared plan and scalar loop;
- [`encode_simd_arm64.go`](../../utf16/encode_simd_arm64.go): NEON path;
- [`encode_simd_amd64.go`](../../utf16/encode_simd_amd64.go): runtime dispatch;
- [`encode_simd_avx2.go`](../../utf16/encode_simd_avx2.go): AVX2 planner and encoders;
- [`encode_simd_avx512.go`](../../utf16/encode_simd_avx512.go): AVX-512 planner and encoder;
- [`encode_fallback.go`](../../utf16/encode_fallback.go): standard-library fallback.
