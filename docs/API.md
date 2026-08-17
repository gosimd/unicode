# API

## UTF-8 package

Import:

```go
import "github.com/gosimd/unicode/utf8"
```

`github.com/gosimd/unicode/utf8` mirrors the API and behaviour of the
repository's Go 1.27rc1 baseline, `unicode/utf8`, and adds whole-buffer
`Encode` and `Decode` conversions that the standard package leaves to language
syntax. The SIMD implementation is an internal detail: results never depend on
CPU architecture or build flags.

| Kind | Names |
| --- | --- |
| Constants | `RuneError`, `RuneSelf`, `MaxRune`, `UTFMax` |
| Validation | `Valid`, `ValidString`, `ValidRune` |
| Decoding | `Decode`, `DecodeRune`, `DecodeRuneInString`, `DecodeLastRune`, `DecodeLastRuneInString` |
| Encoding | `AppendRune`, `Encode`, `EncodeRune`, `RuneLen` |
| Inspection | `FullRune`, `FullRuneInString`, `RuneStart` |
| Counting | `RuneCount`, `RuneCountInString` |

`Valid([]byte)` and `ValidString(string)` use the SIMD validator on supported
builds and otherwise call the standard-library equivalent. The SIMD
`ValidString` path presents the string as a read-only byte slice without
copying. Both forms allocate zero memory.

`RuneCount([]byte)` and `RuneCountInString(string)` validate and count valid
UTF-8 in one SIMD traversal on supported builds. For malformed input they fall
back to `unicode/utf8.RuneCount` semantics, where erroneous and short encodings
count as one width-1 `RuneError` each. `Encode([]rune)` is equivalent to
`string(runes)`, and `Decode(string)` is equivalent to `[]rune(string)`;
both currently use those language conversions directly. Every remaining API
delegates directly to the standard library because its input is at most one
rune or a few bytes and does not benefit from whole-buffer SIMD processing.

The compatibility surface is reviewed when the Go baseline changes. New
exports in `unicode/utf8` are intentionally added here with matching tests;
`Encode` and `Decode` are intentional extensions. Go does not provide a
package-wide re-export mechanism.

## UTF-16 package

Import:

```go
import "github.com/gosimd/unicode/utf16"
```

`github.com/gosimd/unicode/utf16` mirrors the Go 1.27rc1 `unicode/utf16` API:
`IsSurrogate`, `DecodeRune`, `EncodeRune`, `RuneLen`, `Encode`, `AppendRune`,
and `Decode`. On arm64 with `GOEXPERIMENT=simd`, `Decode` uses NEON for
eight-code-unit chunks with no surrogates, and `Encode` narrows eight-rune
chunks when every rune is a valid BMP code point outside the surrogate range.
On amd64, both operations require AVX2. `Encode` packs 64 low-BMP runes per
unrolled iteration without value checks after its length pass has classified
the input; other BMP input uses checked 16-rune blocks. Valid non-BMP and
mixed text use dedicated vector conversion and table-driven compaction.
Surrogate values and invalid runes use the scalar path, preserving
`unicode/utf16.Encode` semantics and result capacity. Unsupported builds
delegate to the standard library. Compile-time function-type checks keep the
compatibility surface synchronized with the standard-library baseline.

## Root package

Import:

```go
import "github.com/gosimd/unicode"
```

The root package is a smaller convenience facade exposing `Valid`,
`ValidString`, and `RuneCount`. It delegates to `github.com/gosimd/unicode/utf8`.

For the `Valid` implementation details, see [Valid.md](Valid.md).
