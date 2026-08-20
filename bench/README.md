# Performance reports

The Markdown files in this directory record an expected performance level for
specific machines and revisions. They are generated locally by
`cmd/benchreport` and are intended to be committed with the repository.

## Matrix

The report contains these operations:

- `utf8.Valid`
- `utf8.RuneCount`
- `utf8.Encode-full` and `utf8.Encode-core`
- `utf8.Decode-full` and `utf8.Decode-core`
- `utf16.Encode-full` and `utf16.Encode-core`
- `utf16.Decode-full` and `utf16.Decode-core`

Each operation uses four approximately 64 KiB inputs: `ascii-only`, `mixed`,
`russian`, and `chinese`. Mixed text includes English, Russian, Chinese, and an
emoji so UTF-16 measurements also exercise surrogate handling.

Time is normalized by decoded Unicode code points. Throughput follows the
existing Go benchmarks: UTF-8 input bytes for UTF-8 validation, counting, and
decoding; 4-byte Go `rune` input for UTF-8 and UTF-16 Encode; and 2-byte UTF-16
input for Decode. Consequently throughput values from different operations are
not interchangeable.

`-full` calls the public function and includes output allocation. `-core`
reuses a caller-owned output buffer. The UTF-8 SIMD core columns isolate the
NEON encoder or decoder after their planning pass; the equivalent stdlib core
columns use caller-owned versions of the scalar conversion loops. The generator
rejects a core result if either gosimd or stdlib reports a timed allocation.
On platforms without the ARM64 UTF-8 conversion backend, the gosimd core row
uses the same scalar caller-buffer loop as the stdlib core row. Since
`unicode/utf16` has no caller-output API, its core columns use
benchmark-local copies of the Go 1.27rc1 Encode and Decode loops; those columns
are a fair algorithm comparison, not a public stdlib API call.

## Generate a report

On macOS or Linux:

```sh
make bench-report
```

The default filename is based on the platform, for example
`bench/results-darwin-arm64.md` or `bench/results-linux-amd64.md`. Use an
explicit hardware-oriented filename when retaining several results for one
platform:

```sh
make bench-report REPORT_OUTPUT=bench/hetzner-ax2-xeon.md
```

Without `make`, including from Windows PowerShell, invoke the project toolchain
directly:

```powershell
$env:GOEXPERIMENT = 'simd'
& go run .\cmd\benchreport -count 5 -benchtime 1s -output bench\windows-i3-8100t.md
```

The generator runs the UTF-8 and UTF-16 packages sequentially so their
benchmarks do not contend with each other. CPU name and frequency are read from
`sysctl` on macOS, `/proc/cpuinfo` on Linux, and `Win32_Processor` on Windows.
Frequency reporting is best effort because some systems, notably Apple Silicon,
do not expose a stable nominal clock. The active SIMD backend is detected from
the same Go SIMD runtime feature checks used by the library. Any metadata can
be corrected explicitly with `-cpu`, `-frequency`, or `-simd`.

For a publishable result, use the same revision and Go toolchain on every
machine, close background workloads, use a stable power/performance mode, and
retain the default five 1-second samples. A short smoke run can use
`-count 1 -benchtime 20ms`, but should not be committed as an expected
performance baseline.
