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
arm64 with NEON or amd64 with AVX2 when `GOEXPERIMENT=simd` is enabled. Its
other functions are standard-library facades.

## Validation dispatch

```text
utf.Valid([]byte) -> gosimd/unicode/utf8.Valid([]byte)
utf.RuneCount([]byte) -> gosimd/unicode/utf8.RuneCount([]byte)

gosimd/unicode/utf8.Valid([]byte)
gosimd/unicode/utf8.RuneCount([]byte)
        |
        +-- arm64 + GOEXPERIMENT=simd -> NEON implementation
        +-- amd64 + GOEXPERIMENT=simd + AVX2 -> SIMD implementation
        +-- otherwise -> unicode/utf8.Valid
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
| `rune_count_simd.go` | SIMD `RuneCount`, `RuneCountInString`, and counting loop. |
| `continuation_count_*.go` | Architecture-specific continuation-mask reduction for rune counting. |
| `utf8_simd_common.go` | Shared chunk sizes, byte classification, scalar tail state, and masks. |
| `valid_predicates_arm64.go` | NEON fused validation predicate and incomplete-sequence carry. |
| `ascii_arm64.go` | NEON ASCII detector using a signed minimum. |
| `valid_predicates_amd64.go` | AVX2 validation predicate and incomplete-sequence carry. |
| `ascii_amd64.go` | amd64 ASCII detector. |
| `lookup_*`, `zero_*` | Architecture-specific table lookup and zero reduction helpers. |
| `valid_fallback.go`, `rune_count_fallback.go` | Non-SIMD standard-library fallbacks. |

The common loop works on four 16-byte vectors (64 bytes) where possible. It
uses a cheap ASCII path; a non-ASCII block is checked as four adjacent UTF-8
chunks. The previous chunk and the incomplete-sequence marker preserve state at
block boundaries. A final tail shorter than 16 bytes is validated with a small
scalar state machine.

See [docs/Valid.md](docs/Valid.md) for the detailed algorithm.

## SIMD UTF-16 decoding

`utf16.Decode` processes eight `uint16` code units at a time. A vector range
test first rejects any chunk containing `0xD800..0xDFFF`. Clean chunks are
zero-extended into `rune` values: NEON widens two groups of four lanes, while
AVX2 widens all eight lanes. A chunk with any surrogate is decoded by the
scalar state machine one code unit at a time, so a high surrogate at a vector
boundary, valid pairs, and malformed sequences have the exact
`unicode/utf16.Decode` result. Unsupported builds, including amd64 without
AVX2, delegate to the standard library.

## SIMD rune counting

`RuneCount` validates and counts valid input in one traversal. The same
64-byte/16-byte shapes and boundary carry as `Valid` establish that each
continuation byte belongs to a well-formed rune. The count is then the number
of bytes minus the SIMD-popcounted continuation bytes. If validation fails, it
falls back to `unicode/utf8.RuneCount`, which preserves its requirement that
each malformed or truncated byte is a width-1 error rune.

## Correctness and performance policy

- `unicode/utf8` is the behavioural baseline.
- SIMD changes require regular tests, fuzzing against the standard library,
  and a fallback build check.
- Benchmark claims include input shape, architecture, toolchain, and
  `-benchmem`; they are observations, not API guarantees.
- Architecture-specific changes stay behind build tags and must preserve the
  same public result on every supported platform.
