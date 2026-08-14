# SIMD algorithm for `github.com/gosimd/unicode/utf8.Valid`

`github.com/gosimd/unicode/utf8.Valid(p []byte)` returns the same result as
`unicode/utf8.Valid` without allocating. The SIMD implementation is enabled only with
`GOEXPERIMENT=simd` on `arm64` or `amd64`. Unsupported builds and amd64 hosts
without AVX2 call `unicode/utf8.Valid` directly.

The root package's `utf.Valid` delegates to this implementation, so it uses
SIMD whenever this package selects the SIMD path. In SIMD-enabled builds,
`ValidString` creates a read-only, zero-copy byte view of its string and calls
the same validator.

## Processing shape

The ARM64 loop and the amd64 baseline load 64 bytes as four 16-byte vectors:

```text
chunk0  chunk1  chunk2  chunk3
  16 B    16 B    16 B    16 B
```

ASCII-only blocks use a cheap detector. Non-ASCII blocks are checked as four
adjacent chunks, passing the preceding chunk to the next check. This baseline
also handles initial amd64 inputs shorter than 512 bytes.

After 64-byte blocks, the implementation processes any remaining complete
16-byte chunks. A final tail of 1 to 15 bytes uses a scalar UTF-8 state machine
because it is too short to load as a vector safely.

For `Valid`, amd64 AVX2 and AVX-512 first scan independent 512-byte windows for
ASCII. A dirty AVX2 window is validated as sixteen 32-byte vectors; AVX-512
uses eight 64-byte vectors. Both wide paths accumulate vector errors across the
window and reduce them once. Later independent ASCII windows can return to the
cheap scan path.

## UTF-8 constraints

The validator rejects all conditions rejected by `unicode/utf8.Valid`:

- unexpected continuation bytes;
- missing continuation bytes and truncated sequences;
- invalid leading bytes (`C0`, `C1`, and `F5` through `FF`);
- overlong forms, including `E0 80..9F` and `F0 80..8F`;
- surrogate encodings (`ED A0..BF`);
- code points beyond `U+10FFFF` (`F4 90..BF`).

## Chunk boundaries and carry

For a chunk `chunk` and its predecessor `prev`, three byte-aligning extracts
provide the preceding one, two, and three positions. They align the possible
leading byte with each byte that could be its first continuation byte.

```text
prev1 = concat(prev, chunk) shifted by 15 bytes
prev2 = concat(prev, chunk) shifted by 14 bytes
prev3 = concat(prev, chunk) shifted by 13 bytes
```

The continuation requirements from those aligned vectors are combined with the
actual continuation-byte classes. A non-zero error vector means that the chunk
is invalid.

At the end of a full 128-bit chunk, `incompleteSIMDChunk` examines only lanes
13, 14, and 15. The wide validators retain the final three bytes explicitly.
Both representations mark a 3-, 2-, or 1-byte continuation requirement that
runs past the processed region. The next block, or end of input, must consume
that state; otherwise the input is truncated.

The scalar tail reconstructs state from the final three bytes only when a tail
is present. This keeps the common full-block path entirely vector-based.

## `RuneCount`

`RuneCount` uses the same validation predicates and boundary carry as `Valid`,
but fuses counting into that traversal. For a well-formed UTF-8 buffer, each
continuation byte belongs to a preceding rune, so the result is
`len(p) - continuationCount`. NEON and AVX2 reduce continuation masks in their
16-byte loops. On AVX-512, the lookup validator already produces a vector with
`0x80` at every required continuation position. `VPSADBW` accumulates those
bytes into eight 64-bit lanes while validating each 64-byte vector; the lanes
are reduced once after the full 512-byte windows have been processed.

Malformed input cannot use that identity: `unicode/utf8.RuneCount` treats each
erroneous or short byte as a width-1 error rune. If the SIMD validator finds an
error, `RuneCount` restarts with `unicode/utf8.RuneCount` to retain exactly
those semantics. The zero-copy `RuneCountInString` path shares this logic.

## ARM64 / NEON

ARM64 uses the fused algorithm derived from simdutf's error-vector approach.
For each 16-byte chunk it:

1. aligns predecessor bytes with `VEXT`-style extracts;
2. obtains special-case error bits from three 16-entry nibble lookup tables;
3. computes continuation requirements with unsigned saturating subtraction;
4. combines the requirement and special-case vectors with XOR.

The three lookup tables jointly encode leading-byte validity, continuation
validity, and the `E0`, `ED`, `F0`, and `F4` second-byte boundary rules. This
avoids executing independent predicate chains for each condition.

The ARM64 ASCII detector ORs the four vectors, reinterprets the result as
signed `int8`, and reduces with a signed minimum. ASCII bytes are non-negative;
any byte with bit 7 set is negative. This generates `VSMINV` and avoids the
previous explicit `0x80` broadcast and mask.

## AMD64

AMD64 requires AVX2 at runtime. Inputs shorter than 512 bytes retain the
128-bit baseline predicate, which builds actual- and expected-continuation
masks and checks invalid leading bytes independently.

For larger inputs, AVX2 scans 512-byte windows and validates dirty windows as
sixteen native 32-byte vectors. `VPERM2I128` prepares the predecessor of each
128-bit shuffle group, and three grouped `VPALIGNR` operations align the prior
one, two, and three bytes. The same three-table lookup and
saturating-subtraction model as NEON encodes the UTF-8 constraints. Errors are
accumulated in a YMM register and reduced once per window with `VPTEST`. An
explicit three-byte tail preserves state between windows and into the scalar
remainder.

On AVX-512 hosts, the validator first tests every 512-byte window for ASCII.
Clean windows are accepted immediately; a window containing a non-ASCII byte is
then checked as eight native 64-byte AVX-512 vectors. The wide predicate uses
the same three-nibble lookup and saturating-subtraction model as ARM64, with a
single accumulated error reduction per 512-byte window. A 64-bit permutation
prepares the predecessor of each 128-bit shuffle group so sequences crossing
either a group or vector boundary remain validatable. This lets a later ASCII
region return to the cheap path after a rare emoji.

`RuneCount` uses the same AVX-512 lookup loop. It accumulates the existing
expected-continuation bytes with `VPSADBW` and `VPADDQ`, avoiding per-vector
mask extraction and scalar population counts. Inputs shorter than one
512-byte window retain the AVX2 path, and the final shorter tail is completed
with the shared scalar UTF-8 state machine.

## Performance notes

Performance depends on input composition and size. For example, on an Apple
M5 with Go 1.27rc1, the ARM64 SIMD path measured about 10.3 GB/s on the
repository's `mixed_64KB` benchmark. Its ASCII path has lower fixed call cost
than the standard library but a higher steady-state per-byte cost on long,
pure-ASCII buffers; the two can therefore cross over around a few KiB.

On an Intel Core i3-8100T at 3.10 GHz with Go 1.27rc1, the AVX2 path measured
about 8.2 GB/s on `mixed_64KB`, 8.4 GB/s on `dense_non_ascii`, and 41 GB/s on
the 64 KiB sparse-emoji input. These are steady-state, allocation-free
measurements with 512-byte window processing.

These measurements are diagnostic snapshots, not an API guarantee. Re-run
benchmarks with `-benchmem` after changing either the algorithm or the Go
toolchain.
