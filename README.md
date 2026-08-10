# gosimd/utf

`gosimd` is a collection of SIMD implementations for important algorithms,
primarily algorithms that process text and text-adjacent encodings.
`github.com/gosimd/unicode` is the UTF-encoding member of that collection: it
provides correct, allocation-free SIMD implementations of UTF algorithms.

The current focus is UTF-8 validation. UTF-16 algorithms are planned next.
Other `gosimd` libraries will cover domains such as JSON and general encoding.

## Current UTF-8 implementation

The active SIMD implementation is `Valid([]byte)` in
`github.com/gosimd/unicode/simd/unicode/utf8`:

```go
package main

import (
	"fmt"

	"github.com/gosimd/unicode/simd/unicode/utf8"
)

func main() {
	p := []byte("hello, \u4e16\u754c")
	fmt.Println(utf8.Valid(p)) // true
}
```

Its result is identical to `unicode/utf8.Valid`. The implementation selects
SIMD only where it is supported and otherwise uses a safe standard-library
fallback.

The root package provides standard-library-compatible `Valid` and `ValidString` facades through `simd/unicode/utf8`. `Valid([]byte)` can therefore use the SIMD implementation, while `ValidString` currently follows the standard library. See
[docs/API.md](docs/API.md) for the full current API and
[docs/Valid.md](docs/Valid.md) for the SIMD algorithm.

## Architecture and platform support

The SIMD implementation package is built with Go's `simd` experiment on
`arm64` and `amd64`.

- `arm64` uses NEON and a table-driven, fused UTF-8 validator.
- `amd64` requires AVX2 at runtime; machines without AVX2 use the standard
  library fallback.
- Other architectures, and builds without `GOEXPERIMENT=simd`, use the pure Go
  fallback.

The algorithm, boundary handling, and platform-specific choices are documented
in [docs/Valid.md](docs/Valid.md). The repository layout and design boundaries
are in [ARCHITECTURE.md](ARCHITECTURE.md).

## Development

This repository is configured for Go 1.27rc1 via the local toolchain at
`../.tools/go1.27rc1`.

```sh
make build
make test
make bench
make profile
make profile-cpu
make profile-mem
```

### UTF-8 validation comparison

`make bench-utf8-report BENCH_COUNT=10` runs the same valid-input matrix as
[`rusticstuff/simdutf8`](https://github.com/rusticstuff/simdutf8/tree/main/bench):
Latin, Cyrillic, Chinese, and emoji input, plus empty input and a 64 KiB input
with an invalid first byte. The target sizes are 2 B, 8 B, 64 B, 512 B, 4 KiB,
64 KiB, and 128 KiB. A multibyte sample may be up to three bytes larger than
its target so that it ends at a valid UTF-8 boundary.

It writes the raw Go benchmark output to `bench/valid.txt` and a standalone
`bench/valid.html` table. The table has gosimd and standard-library columns;
the smaller median `ns/op` is highlighted green. Both generated files are
ignored by Git. `BENCH_TIME` defaults to `1s`; use a shorter value only for a
quick smoke run, for example `BENCH_TIME=100ms BENCH_COUNT=3`. For a result
you intend to publish, record the machine in the report too:

```sh
make bench-utf8-report BENCH_COUNT=10 BENCH_HARDWARE='Apple M4 Max, 16-core CPU'
```

For a direct SIMD test run:

```sh
GOEXPERIMENT=simd ../.tools/go1.27rc1/bin/go test ./...
GOEXPERIMENT=simd ../.tools/go1.27rc1/bin/go test -bench=. -benchmem ./...
```

In VS Code, install the recommended Go extension and use the provided tasks:

- `Go: build`
- `Go: test`
- `Go: benchmark`
- `Go: profile benchmark`
- `Go: pprof CPU`
- `Go: pprof memory`
