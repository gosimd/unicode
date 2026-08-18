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

### Representative 64 KiB performance

Each value is `gosimd input throughput · speedup vs. stdlib`. Results are
medians of five one-second runs with `GOEXPERIMENT=simd` and cover the public
end-to-end APIs, including allocations where applicable. Full measurements,
allocation data, and methodology are published in [bench/](bench/).

#### Apple M5 · macOS/arm64 · NEON

| Operation | ASCII | Mixed | Russian | Chinese |
| --- | --- | --- | --- | --- |
| `utf8.Valid` | 88.1 GB/s · 0.83× | 14.1 GB/s · 4.46× | 14.1 GB/s · 6.20× | 13.9 GB/s · 4.30× |
| `utf8.RuneCount` | 83.9 GB/s · 18.7× | 11.0 GB/s · 5.26× | 10.9 GB/s · 7.39× | 10.9 GB/s · 5.64× |
| `utf8.Encode` | 9.49 GB/s · 5.46× | 2.82 GB/s · 1.82× | 2.18 GB/s · 1.40× | 2.55 GB/s · 2.01× |
| `utf8.Decode` | 4.51 GB/s · 3.76× | 1.55 GB/s · 1.66× | 1.61 GB/s · 2.20× | 2.59 GB/s · 2.54× |
| `utf16.Encode` | 13.4 GB/s · 2.60× | 7.51 GB/s · 2.17× | 13.6 GB/s · 2.64× | 13.6 GB/s · 2.76× |
| `utf16.Decode` | 8.06 GB/s · 6.95× | 4.17 GB/s · 2.98× | 8.31 GB/s · 7.24× | 8.27 GB/s · 8.80× |

Source: [full Apple M5 report](bench/results-darwin-arm64.md).

#### Intel Xeon Skylake · Linux/amd64 · AVX-512

| Operation | ASCII | Mixed | Russian | Chinese |
| --- | --- | --- | --- | --- |
| `utf8.Valid` | 48.3 GB/s · 2.20× | 7.67 GB/s · 10.97× | 7.64 GB/s · 14.13× | 8.07 GB/s · 11.68× |
| `utf8.RuneCount` | 48.3 GB/s · 43.31× | 6.94 GB/s · 17.07× | 6.39 GB/s · 18.66× | 6.55 GB/s · 17.72× |
| `utf8.Encode` | 6.27 GB/s · 15.00× | 1.22 GB/s · 3.33× | 1.08 GB/s · 3.17× | 1.26 GB/s · 4.95× |
| `utf8.Decode` | 0.87 GB/s · 3.05× | 0.55 GB/s · 2.87× | 0.52 GB/s · 3.08× | 1.16 GB/s · 6.32× |
| `utf16.Encode` | 3.15 GB/s · 4.08× | 1.37 GB/s · 1.90× | 2.71 GB/s · 3.84× | 2.74 GB/s · 3.98× |
| `utf16.Decode` | 0.97 GB/s · 5.89× | 0.51 GB/s · 2.80× | 0.87 GB/s · 7.40× | 0.84 GB/s · 7.66× |

Source: [full Linux AVX-512 report](bench/results-linux-amd64.md).

#### Intel Core i3-8100T · Windows/amd64 · AVX2

| Operation | ASCII | Mixed | Russian | Chinese |
| --- | --- | --- | --- | --- |
| `utf8.Valid` | 78.3 GB/s · 2.69× | 9.04 GB/s · 8.91× | 8.96 GB/s · 11.68× | 9.40 GB/s · 9.74× |
| `utf8.RuneCount` | 74.6 GB/s · 24.53× | 8.23 GB/s · 11.10× | 8.12 GB/s · 15.31× | 8.54 GB/s · 13.19× |
| `utf8.Encode` | 9.07 GB/s · 13.43× | 2.36 GB/s · 3.93× | 2.26 GB/s · 4.62× | 2.23 GB/s · 5.11× |
| `utf8.Decode` | 1.63 GB/s · 3.52× | 0.72 GB/s · 2.01× | 0.79 GB/s · 3.13× | 1.45 GB/s · 4.44× |
| `utf16.Encode` | 6.93 GB/s · 4.23× | 3.45 GB/s · 2.29× | 6.56 GB/s · 4.09× | 5.37 GB/s · 3.39× |
| `utf16.Decode` | 2.26 GB/s · 7.01× | 1.80 GB/s · 4.69× | 2.26 GB/s · 7.01× | 2.21 GB/s · 7.68× |

Source: [full Windows AVX2 report](bench/results-windows-amd64.md).

## Documentation

- [API reference](docs/API.md)
- [UTF-8 validation](docs/utf8/Valid.md)
- [UTF-8 encoding](docs/utf8/Encode.md)
- [UTF-8 decoding](docs/utf8/Decode.md)
- [UTF-16 encoding](docs/utf16/Encode.md)
- [UTF-16 decoding](docs/utf16/Decode.md)
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
