# API

## Root package

Import:

```go
import "github.com/gosimd/unicode"
```

### `Valid`

```go
func Valid(p []byte) bool
```

`Valid` reports whether `p` consists entirely of well-formed UTF-8. It delegates to `simd/unicode/utf8.Valid`, which selects SIMD when available, matches `unicode/utf8.Valid`, and does not allocate.

### `ValidString`

```go
func ValidString(s string) bool
```

`ValidString` reports whether `s` consists entirely of well-formed UTF-8. Its
result matches `unicode/utf8.ValidString` and it does not allocate.

It delegates to `simd/unicode/utf8.ValidString`; in supported SIMD builds that implementation presents the string as a read-only byte slice without copying and calls the SIMD-enabled `Valid` path. Other builds use the standard library fallback.

## UTF-8 compatibility package

Import:

```go
import "github.com/gosimd/unicode/simd/unicode/utf8"
```

This package provides the following selected compatibility surface from
`unicode/utf8`:

| Kind | Names |
| --- | --- |
| Constants | `RuneError`, `RuneSelf`, `MaxRune`, `UTFMax` |
| Validation | `Valid`, `ValidString`, `ValidRune` |
| Decoding | `DecodeRune`, `DecodeRuneInString`, `DecodeLastRune`, `DecodeLastRuneInString` |
| Encoding | `AppendRune`, `EncodeRune`, `RuneLen` |
| Inspection | `FullRune`, `FullRuneInString`, `RuneStart` |
| Counting | `RuneCount`, `RuneCountInString` |

`Valid([]byte)` and `ValidString(string)` in this package can use SIMD as
described in [Valid.md](Valid.md). All other listed functions delegate to the
Go standard library and follow its documented behaviour.

## Compatibility and portability

The public result of every API is independent of CPU architecture and build
flags. In particular, callers of the SIMD package must not need to detect AVX2,
NEON, or Go's SIMD experiment; its `Valid` implementation chooses the
available path internally.

For the `Valid` implementation details, see [Valid.md](Valid.md).
