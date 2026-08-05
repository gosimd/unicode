# Agent Instructions

This repository is `github.com/gosimd/utf`: a Go library for fast, correct
operations on UTF encodings. The first implementation target is UTF-8.

## Project Goals

- Build an idiomatic Go API for UTF operations. Public names should describe
  text behavior, not SIMD machinery.
- Keep SIMD as an implementation detail. Do not expose vector types, masks,
  CPU feature flags, or architecture-specific concepts in the public API.
- Prioritize correctness and portability before benchmark wins.
- Match Go standard library semantics wherever an equivalent exists.
- Keep the library usable in ordinary Go projects: no cgo requirement, no
  heavyweight dependencies, and no surprising global state.

## Public API Guidelines

- Prefer small functions with clear contracts, such as `Valid`, `ValidString`,
  `IsASCII`, `IsASCIIString`, and `FirstInvalid`.
- Public functions must behave the same on every supported architecture.
- Avoid allocations in validation, ASCII scanning, and byte classification paths.
- Do not add new exported API without updating README examples and tests.
- If behavior intentionally differs from `unicode/utf8`, document the difference
  explicitly and add tests for it.

## Implementation Guidelines

- Always provide a pure Go fallback implementation.
- Keep architecture-specific code behind build tags and small internal packages.
  Suggested package boundaries are `internal/simd` and `internal/cpu` when they
  become useful.
- Runtime CPU feature detection must fall back safely on unsupported machines.
- Keep unsafe code isolated and justified. Do not use `unsafe` in public-facing
  code unless there is a clear measured need.
- Assembly is acceptable only when paired with a Go reference implementation and
  shared conformance tests.
- Prefer simple, readable scalar code before introducing vectorized versions.

## Correctness Requirements

- Use `unicode/utf8` as the behavioral baseline for UTF-8 validation.
- Cover at least these cases in tests:
  - empty input;
  - pure ASCII;
  - valid 2-byte, 3-byte, and 4-byte sequences;
  - truncated sequences;
  - stray continuation bytes;
  - overlong encodings;
  - surrogate halves;
  - code points above `U+10FFFF`;
  - invalid bytes at the beginning, middle, and end of input.
- Add fuzz tests for validators and invalid-offset helpers when those APIs
  exist.
- Table-driven tests should exercise both `[]byte` and `string` entry points
  when both are available.

## Benchmarks

- Do not claim a speedup without running benchmarks and describing the input.
- Compare against the Go standard library where possible.
- Benchmark realistic shapes:
  - small strings;
  - medium buffers;
  - large buffers;
  - pure ASCII;
  - mostly ASCII with occasional multibyte runes;
  - dense non-ASCII;
  - invalid input early and late in the buffer.
- Include `-benchmem` when reporting benchmark results.
- Performance changes must not weaken validation semantics.

## Development Commands

Run these after the Go module exists:

```sh
go test ./...
go test -bench=. -benchmem ./...
go vet ./...
```

Use `gofmt` on every changed Go file.

## Repository Hygiene

- Keep the module path as `github.com/gosimd/utf` unless the maintainer decides
  otherwise.
- Keep the MIT license intact.
- Do not vendor dependencies unless there is a strong reason.
- Do not commit generated binaries, local benchmark logs, coverage output, or
  editor metadata.
- Update the README whenever public behavior, package layout, or supported
  platforms change.
- Keep changes focused. Avoid unrelated refactors in the same patch.

## Agent Workflow

- Inspect the current repository state before editing.
- Preserve user changes. Do not reset, overwrite, or revert unrelated work.
- If tests or benchmarks cannot be run, say exactly what was not run and why.
- For SIMD, assembly, or CPU-detection changes, mention the architecture used
  for verification.
- Be precise about performance: report measured results, not expectations.
