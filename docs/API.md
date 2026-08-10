# API

## UTF-8 package

Import:

```go
import "github.com/gosimd/unicode/utf8"
```

`github.com/gosimd/unicode/utf8` is the public UTF-8 package. It mirrors the
API and behaviour of the repository's Go 1.27rc1 baseline, `unicode/utf8`, so
a caller can replace that import without changing call sites. The SIMD
implementation is an internal detail: results never depend on CPU architecture
or build flags.

| Kind | Names |
| --- | --- |
| Constants | `RuneError`, `RuneSelf`, `MaxRune`, `UTFMax` |
| Validation | `Valid`, `ValidString`, `ValidRune` |
| Decoding | `DecodeRune`, `DecodeRuneInString`, `DecodeLastRune`, `DecodeLastRuneInString` |
| Encoding | `AppendRune`, `EncodeRune`, `RuneLen` |
| Inspection | `FullRune`, `FullRuneInString`, `RuneStart` |
| Counting | `RuneCount`, `RuneCountInString` |

`Valid([]byte)` and `ValidString(string)` use the SIMD validator on supported
builds and otherwise call the standard-library equivalent. The SIMD
`ValidString` path presents the string as a read-only byte slice without
copying. Both forms allocate zero memory.

`RuneCount([]byte)` and `RuneCountInString(string)` validate and count valid
UTF-8 in one SIMD traversal on supported builds. For malformed input they fall
back to `unicode/utf8.RuneCount` semantics, where erroneous and short encodings
count as one width-1 `RuneError` each. Every other API delegates directly to
the standard library because its input is at most one rune or a few bytes and
does not benefit from whole-buffer SIMD processing.

The compatibility surface is reviewed when the Go baseline changes. New
exports in `unicode/utf8` are intentionally added here with matching tests;
Go does not provide a package-wide re-export mechanism.

## Root package

Import:

```go
import "github.com/gosimd/unicode"
```

The root package is a smaller convenience facade exposing `Valid`,
`ValidString`, and `RuneCount`. It delegates to `github.com/gosimd/unicode/utf8`.

For the `Valid` implementation details, see [Valid.md](Valid.md).
