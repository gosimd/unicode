# Unicode performance on Apple M5

This report records the expected performance level for this machine. Results are medians; higher throughput and a larger speedup are better, while lower time per character is better.

## Environment

| Parameter | Value |
|---|---|
| CPU | Apple M5 |
| Frequency | not reported |
| Active SIMD backend | ARM NEON |
| Logical CPUs | 10 |
| Platform | `darwin/arm64` |
| Go | `go1.27rc1-X:simd` with `GOEXPERIMENT=simd` |
| Git revision | `64d1d4cb9e92+dirty` |
| Generated (UTC) | `2026-08-14T15:25:06Z` |
| Sampling | median of 5 samples, `-benchtime=1s` |

## Workloads

Every row uses an approximately 64 KiB input working set. `ascii-only` is English ASCII; `mixed` combines English, Russian, Chinese, and emoji; `russian` and `chinese` contain only their named scripts. Repetition ends only at a valid encoding boundary.

For UTF-8, throughput counts UTF-8 input bytes. For UTF-16 Encode it counts the 4-byte Go `rune` input, and for Decode it counts the 2-byte UTF-16 input, matching the package benchmarks. A character means one decoded Unicode code point. `-full` calls the public API and includes output allocation; `-core` reuses caller-owned output but includes length/planning and conversion work.

## UTF-8

| API | Scenario | Input | gosimd time/char | gosimd throughput | gosimd allocation | stdlib time/char | stdlib throughput | stdlib allocation | Speedup |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `utf8.Valid` | ascii-only | 64.03 KiB / 65565 chars | 0.012 ns | 84.33 GB/s | 0 B, 0 allocs | **0.009 ns** | **105.66 GB/s** | 0 B, 0 allocs | 0.80× (-20.2%) |
| `utf8.Valid` | mixed | 64.01 KiB / 43125 chars | **0.112 ns** | **13.59 GB/s** | 0 B, 0 allocs | 0.576 ns | 2.64 GB/s | 0 B, 0 allocs | 5.15× (+415.2%) |
| `utf8.Valid` | russian | 64.00 KiB / 36410 chars | **0.132 ns** | **13.66 GB/s** | 0 B, 0 allocs | 0.988 ns | 1.82 GB/s | 0 B, 0 allocs | 7.50× (+649.6%) |
| `utf8.Valid` | chinese | 64.00 KiB / 21846 chars | **0.221 ns** | **13.60 GB/s** | 0 B, 0 allocs | 1.156 ns | 2.59 GB/s | 0 B, 0 allocs | 5.24× (+424.3%) |
| `utf8.RuneCount` | ascii-only | 64.03 KiB / 65565 chars | **0.012 ns** | **83.77 GB/s** | 0 B, 0 allocs | 0.228 ns | 4.39 GB/s | 0 B, 0 allocs | 19.07× (+1807.1%) |
| `utf8.RuneCount` | mixed | 64.01 KiB / 43125 chars | **0.138 ns** | **11.00 GB/s** | 0 B, 0 allocs | 0.724 ns | 2.10 GB/s | 72.00 KiB, 1 allocs | 5.24× (+423.9%) |
| `utf8.RuneCount` | russian | 64.00 KiB / 36410 chars | **0.163 ns** | **11.04 GB/s** | 0 B, 0 allocs | 1.216 ns | 1.48 GB/s | 72.00 KiB, 1 allocs | 7.45× (+645.4%) |
| `utf8.RuneCount` | chinese | 64.00 KiB / 21846 chars | **0.273 ns** | **11.00 GB/s** | 0 B, 0 allocs | 1.499 ns | 2.00 GB/s | 72.00 KiB, 1 allocs | 5.49× (+449.3%) |

## UTF-16

| API | Scenario | Input | gosimd time/char | gosimd throughput | gosimd allocation | stdlib time/char | stdlib throughput | stdlib allocation | Speedup |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `utf16.Encode-full` | ascii-only | 64.00 KiB / 16384 chars | **0.299 ns** | **13.37 GB/s** | 32.00 KiB, 1 allocs | 0.775 ns | 5.16 GB/s | 32.00 KiB, 1 allocs | 2.59× (+158.9%) |
| `utf16.Encode-full` | mixed | 64.00 KiB / 16384 chars | **0.499 ns** | **8.01 GB/s** | 40.00 KiB, 1 allocs | 0.841 ns | 4.76 GB/s | 40.00 KiB, 1 allocs | 1.68× (+68.4%) |
| `utf16.Encode-full` | russian | 64.00 KiB / 16384 chars | **0.297 ns** | **13.48 GB/s** | 32.00 KiB, 1 allocs | 0.776 ns | 5.16 GB/s | 32.00 KiB, 1 allocs | 2.61× (+161.3%) |
| `utf16.Encode-full` | chinese | 64.00 KiB / 16384 chars | **0.296 ns** | **13.52 GB/s** | 32.00 KiB, 1 allocs | 0.776 ns | 5.16 GB/s | 32.00 KiB, 1 allocs | 2.62× (+162.3%) |
| `utf16.Encode-core` | ascii-only | 64.00 KiB / 16384 chars | **0.209 ns** | **19.10 GB/s** | 0 B, 0 allocs | 0.925 ns | 4.33 GB/s | 0 B, 0 allocs | 4.41× (+341.4%) |
| `utf16.Encode-core` | mixed | 64.00 KiB / 16384 chars | **0.397 ns** | **10.08 GB/s** | 0 B, 0 allocs | 0.831 ns | 4.81 GB/s | 0 B, 0 allocs | 2.09× (+109.4%) |
| `utf16.Encode-core` | russian | 64.00 KiB / 16384 chars | **0.210 ns** | **19.09 GB/s** | 0 B, 0 allocs | 0.927 ns | 4.32 GB/s | 0 B, 0 allocs | 4.42× (+342.3%) |
| `utf16.Encode-core` | chinese | 64.00 KiB / 16384 chars | **0.210 ns** | **19.09 GB/s** | 0 B, 0 allocs | 0.922 ns | 4.34 GB/s | 0 B, 0 allocs | 4.40× (+340.0%) |
| `utf16.Decode-full` | ascii-only | 64.07 KiB / 32805 chars | **0.245 ns** | **8.16 GB/s** | 136.00 KiB, 1 allocs | 1.722 ns | 1.16 GB/s | 657.63 KiB, 17 allocs | 7.03× (+602.6%) |
| `utf16.Decode-full` | mixed | 64.04 KiB / 31525 chars | **0.508 ns** | **4.09 GB/s** | 136.00 KiB, 1 allocs | 1.569 ns | 1.33 GB/s | 489.63 KiB, 16 allocs | 3.09× (+208.9%) |
| `utf16.Decode-full` | russian | 64.02 KiB / 32780 chars | **0.231 ns** | **8.65 GB/s** | 136.00 KiB, 1 allocs | 1.728 ns | 1.16 GB/s | 657.63 KiB, 17 allocs | 7.47× (+647.3%) |
| `utf16.Decode-full` | chinese | 64.02 KiB / 32780 chars | **0.232 ns** | **8.60 GB/s** | 136.00 KiB, 1 allocs | 1.722 ns | 1.16 GB/s | 657.63 KiB, 17 allocs | 7.41× (+640.8%) |
| `utf16.Decode-core` | ascii-only | 64.07 KiB / 32805 chars | **0.058 ns** | **34.75 GB/s** | 0 B, 0 allocs | 0.475 ns | 4.21 GB/s | 0 B, 0 allocs | 8.25× (+725.1%) |
| `utf16.Decode-core` | mixed | 64.04 KiB / 31525 chars | **0.323 ns** | **6.45 GB/s** | 0 B, 0 allocs | 0.522 ns | 3.98 GB/s | 0 B, 0 allocs | 1.62× (+61.9%) |
| `utf16.Decode-core` | russian | 64.02 KiB / 32780 chars | **0.058 ns** | **34.78 GB/s** | 0 B, 0 allocs | 0.466 ns | 4.29 GB/s | 0 B, 0 allocs | 8.11× (+710.9%) |
| `utf16.Decode-core` | chinese | 64.02 KiB / 32780 chars | **0.058 ns** | **34.76 GB/s** | 0 B, 0 allocs | 0.470 ns | 4.26 GB/s | 0 B, 0 allocs | 8.16× (+716.3%) |

## Reproduce

```text
GOEXPERIMENT=simd ../.tools/go1.27rc1/bin/go test -run=^$ -bench=^BenchmarkReport$ -benchmem -benchtime=1s -count=5 ./utf8
GOEXPERIMENT=simd ../.tools/go1.27rc1/bin/go test -run=^$ -bench=^BenchmarkReport$ -benchmem -benchtime=1s -count=5 ./utf16
```
