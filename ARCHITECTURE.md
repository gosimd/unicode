# Architecture

## Design boundaries

The root package, `github.com/gosimd/utf`, is the stable user-facing surface.
It exposes text operations and must not expose vector types, masks, CPU feature
checks, or architecture-specific behaviour. Its current validation methods
delegate directly to `unicode/utf8`.

`simd/unicode/utf8` contains the UTF-8 implementation. Its API mirrors a
selected part of `unicode/utf8`, with `Valid([]byte)` as its optimized path.
Callers that explicitly import this package can use the SIMD implementation.

## Validation dispatch

```text
utf.Valid([]byte) -> unicode/utf8.Valid([]byte)

simd/unicode/utf8.Valid([]byte)
        |
        +-- arm64 + GOEXPERIMENT=simd -> NEON implementation
        +-- amd64 + GOEXPERIMENT=simd + AVX2 -> SIMD implementation
        +-- otherwise -> unicode/utf8.Valid
```

The fallback is part of the correctness contract, not an exceptional path.
Every implementation must return the same result as `unicode/utf8.Valid`.

Both `utf.ValidString` and `simd/unicode/utf8.ValidString` currently delegate
to `unicode/utf8.ValidString`; they do not convert a string to `[]byte` and do
not allocate.

## SIMD implementation layout

| File group | Responsibility |
| --- | --- |
| `valid_simd.go` | Common block loop, final scalar tail, and cross-chunk state. |
| `valid_predicates_arm64.go` | NEON fused validation predicate and incomplete-sequence carry. |
| `ascii_arm64.go` | NEON ASCII detector using a signed minimum. |
| `valid_predicates_amd64.go` | AVX2 validation predicate and incomplete-sequence carry. |
| `ascii_amd64.go` | amd64 ASCII detector. |
| `lookup_*`, `zero_*` | Architecture-specific table lookup and zero reduction helpers. |
| `valid_fallback.go` | Non-SIMD `unicode/utf8.Valid` fallback. |

The common loop works on four 16-byte vectors (64 bytes) where possible. It
uses a cheap ASCII path; a non-ASCII block is checked as four adjacent UTF-8
chunks. The previous chunk and the incomplete-sequence marker preserve state at
block boundaries. A final tail shorter than 16 bytes is validated with a small
scalar state machine.

See [docs/Valid.md](docs/Valid.md) for the detailed algorithm.

## Correctness and performance policy

- `unicode/utf8` is the behavioural baseline.
- SIMD changes require regular tests, fuzzing against the standard library,
  and a fallback build check.
- Benchmark claims include input shape, architecture, toolchain, and
  `-benchmem`; they are observations, not API guarantees.
- Architecture-specific changes stay behind build tags and must preserve the
  same public result on every supported platform.
