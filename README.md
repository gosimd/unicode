# gosimd/unicode

`gosimd` is a collection of SIMD implementations for important algorithms,
primarily algorithms that process text and text-adjacent encodings.
`github.com/gosimd/unicode` is the UTF-encoding member of that collection: it
provides correct, allocation-free SIMD implementations of UTF algorithms.

The current focus is UTF-8 validation. A UTF-16-compatible facade is available;
SIMD UTF-16 algorithms are planned next.
Other `gosimd` libraries will cover domains such as JSON and general encoding.

## Current UTF-8 implementation

The public UTF-8 package is `github.com/gosimd/unicode/utf8`. It mirrors the
current `unicode/utf8` API, adds `Encode([]rune)` and `Decode(string)` as
language-conversion conveniences, and accelerates selected whole-buffer
operations:

```go
package main

import (
	"fmt"

	"github.com/gosimd/unicode/utf8"
)

func main() {
	p := []byte("hello, \u4e16\u754c")
	fmt.Println(utf8.Valid(p)) // true
}
```

Its result is identical to `unicode/utf8.Valid`. The implementation selects
SIMD only where it is supported and otherwise uses a safe standard-library
fallback.

The root package provides `Valid`, `ValidString`, and `RuneCount` convenience
facades through `github.com/gosimd/unicode/utf8`. `Valid` and `ValidString`
use the SIMD validator when available; `ValidString` passes the string to it
without copying. `RuneCount` has the same semantics as `unicode/utf8.RuneCount`
and uses a SIMD one-pass validator/counter for valid UTF-8 when supported. See
[docs/API.md](docs/API.md) for the full current API and
[docs/Valid.md](docs/Valid.md) for the SIMD algorithm.

## UTF-16 compatibility package

`github.com/gosimd/unicode/utf16` mirrors `unicode/utf16`, so an existing
standard-library import can be replaced without changing call sites. Its
`Decode` function widens eight-code-unit blocks without surrogates with NEON
on arm64 or AVX2 on amd64 when built with `GOEXPERIMENT=simd`. AVX2 builds do
not require AVX-512; AVX-512-capable AMD64 hosts select a separate path. On
arm64, `Encode` narrows clean eight-rune blocks. AVX2 uses a predicate-free,
64-rune-unrolled path for text below `U+D800` and checked 16-rune blocks for
other BMP text. Valid non-BMP text is converted to surrogate pairs in
32-rune-unrolled vectors; mixed valid text uses table-driven SIMD compaction.
Surrogate values and invalid runes retain the standard-library scalar
behaviour; unsupported builds delegate to the standard library.

## Architecture and platform support

The SIMD implementation package is built with Go's `simd` experiment on
`arm64` and `amd64`.

- `arm64` uses NEON and a table-driven, fused UTF-8 validator.
- `amd64` requires AVX2 at runtime. `Valid` and `RuneCount` use native 256-bit
  validators for dirty 512-byte windows; machines without AVX2 use the standard
  library fallback.
- Other architectures, and builds without `GOEXPERIMENT=simd`, use the pure Go
  fallback.

The algorithm, boundary handling, and platform-specific choices are documented
in [docs/Valid.md](docs/Valid.md). The repository layout and design boundaries
are in [ARCHITECTURE.md](ARCHITECTURE.md).

## Development

This repository selects Go 1.27rc3 through the machine-local `GOSIMD_GO`
environment variable. Set it once to the `go` executable for the current
machine, add its directory to `PATH`, then restart VS Code so its Go extension
inherits the same environment:

```sh
export GOSIMD_GO=/machine-specific/path/to/go1.27rc3/bin/go
export PATH="$(dirname "$GOSIMD_GO"):$PATH"
```

```sh
make build
make test
make bench
make profile
make profile-cpu
make profile-mem
```

### Performance reports

`make bench-report` measures the stable UTF-8 and UTF-16 publication matrix on
the local machine and writes a commit-ready Markdown file under `bench/`. The
matrix covers ASCII-only, mixed, Russian, and Chinese inputs for `utf8.Valid`,
`utf8.RuneCount`, and the full/core variants of `utf16.Encode` and
`utf16.Decode`. Every row compares gosimd with the equivalent standard-library
implementation and reports time per character, input throughput, allocations,
and speedup.

The default report uses the median of five 1-second samples. It detects the CPU
name, frequency when the operating system exposes it, and the active NEON,
AVX2, or AVX-512 backend. See [bench/README.md](bench/README.md) for the exact
workload contract, report filenames, cross-platform commands, and publication
guidance.

For a direct SIMD test run:

```sh
GOEXPERIMENT=simd "$GOSIMD_GO" test ./...
GOEXPERIMENT=simd "$GOSIMD_GO" test -bench=. -benchmem ./...
```

In VS Code, install the recommended Go extension and use the provided tasks:

- `Go: build`
- `Go: test`
- `Go: benchmark`
- `Go: profile benchmark`
- `Go: pprof CPU`
- `Go: pprof memory`

### Disassembly in VS Code

The recommended `golang.go` extension is installed locally. For the optimized
SIMD machine code with its Go source alongside, run **Tasks: Run Task** and
choose `Go: disassemble UTF-8 symbol (SIMD)`. The task asks for a symbol suffix
or Go regular expression; its default `Valid$` selects `utf8.Valid`. The
disassembly opens in VS Code's dedicated integrated-terminal panel. Use
`RuneCount$` to show that exported function instead.

For an interactive debug session, put a breakpoint in a UTF-8 test and start
`Go: debug UTF-8 tests (SIMD, registers)`. When execution stops, use the
Command Palette command **Debug: Open Disassembly View**. That configuration
uses Delve's DAP adapter and shows registers. Delve compiles debug sessions
without optimizations and inlining; use the task above when inspecting
release-style optimized code.

To start every package's tests under Delve, choose
`Go: debug all tests (SIMD, registers)` from the Run and Debug selector. Go
tests are separate executables per package, so this compound starts the root,
`cmd/benchreport`, and `utf8` test sessions together. Select the
stopped session of interest before opening the Disassembly View.
