# Architecture

## Design boundaries

The root package, `github.com/gosimd/unicode`, is the stable user-facing surface.
It exposes text operations and must not expose vector types, masks, CPU feature
checks, or architecture-specific behaviour. Its validation and rune-counting
methods delegate to `simd/unicode/utf8`; `Valid([]byte)` and `RuneCount([]byte)`
therefore select the SIMD path when it is available.

`simd/unicode/utf8` contains the UTF-8 implementation. Its API mirrors a
selected part of `unicode/utf8`, with `Valid([]byte)` and `RuneCount([]byte)`
as its optimized paths. Callers that explicitly import this package can use the
SIMD implementation.

## Validation dispatch

```text
utf.Valid([]byte) -> simd/unicode/utf8.Valid([]byte)
utf.RuneCount([]byte) -> simd/unicode/utf8.RuneCount([]byte)

simd/unicode/utf8.Valid([]byte)
simd/unicode/utf8.RuneCount([]byte)
        |
        +-- arm64 + GOEXPERIMENT=simd -> NEON implementation
        +-- amd64 + GOEXPERIMENT=simd + AVX2 -> SIMD implementation
        +-- otherwise -> unicode/utf8.Valid
```

The fallback is part of the correctness contract, not an exceptional path.
Every implementation must return the same result as its corresponding
`unicode/utf8` function.

`simd/unicode/utf8.ValidString` and `RuneCountInString` create read-only,
zero-copy byte views when the SIMD implementation is available. The root
package exposes `ValidString`; both supported and fallback paths do not
allocate.

## SIMD implementation layout

| File group | Responsibility |
| --- | --- |
| `valid_simd.go` | SIMD `Valid` and `ValidString` entry points. |
| `rune_count_simd.go` | SIMD `RuneCount`, `RuneCountInString`, and counting loop. |
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
