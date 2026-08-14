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

The main loop loads 64 bytes as four 16-byte vectors:

```text
chunk0  chunk1  chunk2  chunk3
  16 B    16 B    16 B    16 B
```

ASCII-only blocks use a cheap detector. Non-ASCII blocks are checked as four
adjacent chunks, passing the preceding chunk to the next check. Passing this
context is necessary because a multi-byte sequence may cross a 16-byte
boundary.

After 64-byte blocks, the implementation processes any remaining complete
16-byte chunks. A final tail of 1 to 15 bytes uses a scalar UTF-8 state machine
because it is too short to load as a vector safely.

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

At the end of a full chunk, `incompleteSIMDChunk` examines only lanes 13, 14,
and 15. Saturating subtraction against thresholds `EF`, `DF`, and `BF` marks a
3-, 2-, or 1-byte continuation requirement that runs past the chunk boundary.
The next block, or end of input, must consume that state; otherwise the input
is truncated.

The scalar tail reconstructs state from the final three bytes only when a tail
is present. This keeps the common full-block path entirely vector-based.

## `RuneCount`

`RuneCount` uses the same block sizes, validation predicates, and boundary
carry as `Valid`, but fuses counting into that traversal. For a well-formed
UTF-8 buffer, each continuation byte belongs to a preceding rune, so the result
is `len(p) - continuationCount`. The continuation predicate is computed as a
SIMD mask and reduced with two 64-bit population counts per 16-byte chunk.

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

AMD64 requires AVX2 at runtime. Its AVX2 predicate is intentionally kept
separate from the ARM64 fused implementation: it builds actual-continuation and
expected-continuation masks, rejects invalid leading bytes, and applies the
four exceptional second-byte checks. The same vector incomplete-sequence marker
is used at full-chunk boundaries.

On AVX-512 hosts, the validator first tests every 512-byte window for ASCII.
Clean windows are accepted immediately; a window containing a non-ASCII byte is
then checked as eight native 64-byte AVX-512 vectors. This lets a later ASCII
region return to the cheap path after a rare emoji, while a continuation that
crosses a window boundary is still checked by the full predicate.

## Performance notes

Performance depends on input composition and size. For example, on an Apple
M5 with Go 1.27rc1, the ARM64 SIMD path measured about 10.3 GB/s on the
repository's `mixed_64KB` benchmark. Its ASCII path has lower fixed call cost
than the standard library but a higher steady-state per-byte cost on long,
pure-ASCII buffers; the two can therefore cross over around a few KiB.

These measurements are diagnostic snapshots, not an API guarantee. Re-run
benchmarks with `-benchmem` after changing either the algorithm or the Go
toolchain.
