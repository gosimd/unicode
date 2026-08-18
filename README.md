# gosimd/unicode

SIMD-accelerated UTF-8 and UTF-16 operations for Go. The packages preserve the
documented behaviour of their Go standard-library counterparts and use a safe
fallback when SIMD is unavailable.

## Packages

- [`github.com/gosimd/unicode/utf8`](https://pkg.go.dev/github.com/gosimd/unicode/utf8)
  provides UTF-8 validation, rune counting, encoding, and decoding.
- [`github.com/gosimd/unicode/utf16`](https://pkg.go.dev/github.com/gosimd/unicode/utf16)
  provides UTF-16 encoding and decoding.

## Quick start

Validate a UTF-8 buffer:

```go
package main

import (
	"fmt"

	"github.com/gosimd/unicode/utf8"
)

func main() {
	p := []byte("hello, 世界")
	fmt.Println(utf8.Valid(p)) // true
}
```

Convert a sequence of runes to UTF-16 and back:

```go
package main

import (
	"fmt"

	"github.com/gosimd/unicode/utf16"
)

func main() {
	encoded := utf16.Encode([]rune("hello, 世界"))
	fmt.Println(string(utf16.Decode(encoded))) // hello, 世界
}
```

## Compatibility and portability

`utf8` mirrors the `unicode/utf8` API and behaviour, with two intentional
whole-buffer extensions:

- `utf8.Encode([]rune)` is equivalent to `string(runes)`.
- `utf8.Decode(string)` is equivalent to `[]rune(s)`.

`utf16` is API-compatible with `unicode/utf16`. Results do not depend on the
CPU architecture or whether SIMD is enabled.

SIMD acceleration is enabled on supported builds with `GOEXPERIMENT=simd`:

- On `arm64`, accelerated paths use NEON.
- On `amd64`, accelerated paths require AVX2 and may select wider AVX-512
  implementations when available.
- Other architectures, builds without the SIMD experiment, and AMD64 hosts
  without AVX2 use the portable fallback.

## Performance

The accelerated operations are designed for whole buffers. Their benefit
depends on the operation, input shape, CPU, and build configuration; SIMD is
not a universal speedup. Benchmark results should always state the input and
whether they include allocation costs.

`make bench-report` produces a reproducible UTF-8 and UTF-16 comparison matrix
against the standard library, including throughput, allocations, and
per-character time. See [bench/README.md](bench/README.md) for the benchmark
contract and reporting guidance.

## Documentation

- [API reference](docs/API.md)
- [UTF-8 validation](docs/Valid.md)
- [UTF-8 encoding](docs/Encode.md)
- [UTF-8 decoding](docs/Decode.md)
- [Architecture and implementation notes](ARCHITECTURE.md)

## Contributing

Run the portable test suite before submitting a change:

```sh
go test ./...
```

For changes to SIMD paths, also test an enabled SIMD build on the relevant
architecture:

```sh
GOEXPERIMENT=simd go test ./...
```

The project uses Go 1.27. See the [Makefile](Makefile) for the complete build,
benchmark, profiling, and formatting commands.

## License

This project is licensed under the [MIT License](LICENSE).
