# Unicode performance on Intel(R) Core(TM) i3-8100T CPU @ 3.10GHz

This report records the expected performance level for this machine. Results are medians; higher throughput and a larger speedup are better, while lower time per character is better.

## Environment

| Parameter | Value |
|---|---|
| CPU | Intel(R) Core(TM) i3-8100T CPU @ 3.10GHz |
| Frequency | 3.10 GHz (maximum reported) |
| Active SIMD backend | AVX2 |
| Logical CPUs | 4 |
| Platform | `windows/amd64` |
| Go | `go1.27rc1-X:simd` with `GOEXPERIMENT=simd` |
| Git revision | `80ff087877c5+dirty` |
| Generated (UTC) | `2026-08-17T19:29:22Z` |
| Sampling | median of 5 samples, `-benchtime=1s` |

## Workloads

Every row uses an approximately 64 KiB input working set. `ascii-only` is English ASCII; `mixed` combines English, Russian, Chinese, and emoji; `russian` and `chinese` contain only their named scripts. Repetition ends only at a valid encoding boundary.

For UTF-8, throughput counts UTF-8 input bytes. For UTF-16 Encode it counts the 4-byte Go `rune` input, and for Decode it counts the 2-byte UTF-16 input, matching the package benchmarks. A character means one decoded Unicode code point. `-full` calls the public API and includes output allocation; `-core` reuses caller-owned output but includes length/planning and conversion work.

## UTF-8

| API | Scenario | Input | gosimd time/char | gosimd throughput | gosimd allocation | stdlib time/char | stdlib throughput | stdlib allocation | Speedup |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `utf8.Valid` | ascii-only | 64.03 KiB / 65565 chars | **0.013 ns** | **78.73 GB/s** | 0 B, 0 allocs | 0.034 ns | 29.13 GB/s | 0 B, 0 allocs | 2.70× (+170.3%) |
| `utf8.Valid` | mixed | 64.01 KiB / 43125 chars | **0.169 ns** | **8.99 GB/s** | 0 B, 0 allocs | 1.495 ns | 1.02 GB/s | 0 B, 0 allocs | 8.84× (+784.1%) |
| `utf8.Valid` | russian | 64.00 KiB / 36410 chars | **0.203 ns** | **8.86 GB/s** | 0 B, 0 allocs | 2.361 ns | 762.37 MB/s | 0 B, 0 allocs | 11.62× (+1062.3%) |
| `utf8.Valid` | chinese | 64.00 KiB / 21846 chars | **0.322 ns** | **9.31 GB/s** | 0 B, 0 allocs | 3.105 ns | 966.21 MB/s | 0 B, 0 allocs | 9.64× (+863.8%) |
| `utf8.RuneCount` | ascii-only | 64.03 KiB / 65565 chars | **0.013 ns** | **74.97 GB/s** | 0 B, 0 allocs | 0.655 ns | 1.53 GB/s | 0 B, 0 allocs | 49.09× (+4809.2%) |
| `utf8.RuneCount` | mixed | 64.01 KiB / 43125 chars | **0.186 ns** | **8.17 GB/s** | 0 B, 0 allocs | 2.043 ns | 744.00 MB/s | 72.00 KiB, 1 allocs | 10.98× (+997.7%) |
| `utf8.RuneCount` | russian | 64.00 KiB / 36410 chars | **0.222 ns** | **8.12 GB/s** | 0 B, 0 allocs | 3.540 ns | 508.42 MB/s | 72.00 KiB, 1 allocs | 15.98× (+1497.9%) |
| `utf8.RuneCount` | chinese | 64.00 KiB / 21846 chars | **0.354 ns** | **8.47 GB/s** | 0 B, 0 allocs | 4.430 ns | 677.22 MB/s | 72.00 KiB, 1 allocs | 12.51× (+1151.0%) |

## UTF-16

| API | Scenario | Input | gosimd time/char | gosimd throughput | gosimd allocation | stdlib time/char | stdlib throughput | stdlib allocation | Speedup |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `utf16.Encode-full` | ascii-only | 64.00 KiB / 16384 chars | **0.587 ns** | **6.81 GB/s** | 32.00 KiB, 1 allocs | 2.482 ns | 1.61 GB/s | 32.00 KiB, 1 allocs | 4.23× (+322.8%) |
| `utf16.Encode-full` | mixed | 64.00 KiB / 16384 chars | **1.231 ns** | **3.25 GB/s** | 40.00 KiB, 1 allocs | 2.625 ns | 1.52 GB/s | 40.00 KiB, 1 allocs | 2.13× (+113.3%) |
| `utf16.Encode-full` | russian | 64.00 KiB / 16384 chars | **0.588 ns** | **6.80 GB/s** | 32.00 KiB, 1 allocs | 2.544 ns | 1.57 GB/s | 32.00 KiB, 1 allocs | 4.33× (+332.7%) |
| `utf16.Encode-full` | chinese | 64.00 KiB / 16384 chars | **0.794 ns** | **5.04 GB/s** | 32.00 KiB, 1 allocs | 2.527 ns | 1.58 GB/s | 32.00 KiB, 1 allocs | 3.18× (+218.4%) |
| `utf16.Encode-core` | ascii-only | 64.00 KiB / 16384 chars | **0.154 ns** | **26.04 GB/s** | 0 B, 0 allocs | 2.275 ns | 1.76 GB/s | 0 B, 0 allocs | 14.81× (+1380.9%) |
| `utf16.Encode-core` | mixed | 64.00 KiB / 16384 chars | **0.802 ns** | **4.98 GB/s** | 0 B, 0 allocs | 2.416 ns | 1.66 GB/s | 0 B, 0 allocs | 3.01× (+201.1%) |
| `utf16.Encode-core` | russian | 64.00 KiB / 16384 chars | **0.152 ns** | **26.39 GB/s** | 0 B, 0 allocs | 2.231 ns | 1.79 GB/s | 0 B, 0 allocs | 14.72× (+1372.4%) |
| `utf16.Encode-core` | chinese | 64.00 KiB / 16384 chars | **0.350 ns** | **11.44 GB/s** | 0 B, 0 allocs | 2.317 ns | 1.73 GB/s | 0 B, 0 allocs | 6.63× (+562.5%) |
| `utf16.Decode-full` | ascii-only | 64.07 KiB / 32805 chars | **0.872 ns** | **2.29 GB/s** | 136.00 KiB, 1 allocs | 6.265 ns | 319.25 MB/s | 657.63 KiB, 17 allocs | 7.18× (+618.4%) |
| `utf16.Decode-full` | mixed | 64.04 KiB / 31525 chars | **1.190 ns** | **1.75 GB/s** | 136.00 KiB, 1 allocs | 5.657 ns | 367.67 MB/s | 489.63 KiB, 16 allocs | 4.75× (+375.5%) |
| `utf16.Decode-full` | russian | 64.02 KiB / 32780 chars | **0.890 ns** | **2.25 GB/s** | 136.00 KiB, 1 allocs | 6.237 ns | 320.69 MB/s | 657.63 KiB, 17 allocs | 7.01× (+601.1%) |
| `utf16.Decode-full` | chinese | 64.02 KiB / 32780 chars | **0.909 ns** | **2.20 GB/s** | 136.00 KiB, 1 allocs | 7.503 ns | 266.55 MB/s | 657.63 KiB, 17 allocs | 8.25× (+725.2%) |
| `utf16.Decode-core` | ascii-only | 64.07 KiB / 32805 chars | **0.142 ns** | **14.04 GB/s** | 0 B, 0 allocs | 0.993 ns | 2.01 GB/s | 0 B, 0 allocs | 6.97× (+597.4%) |
| `utf16.Decode-core` | mixed | 64.04 KiB / 31525 chars | **0.381 ns** | **5.46 GB/s** | 0 B, 0 allocs | 1.176 ns | 1.77 GB/s | 0 B, 0 allocs | 3.09× (+208.8%) |
| `utf16.Decode-core` | russian | 64.02 KiB / 32780 chars | **0.144 ns** | **13.93 GB/s** | 0 B, 0 allocs | 0.982 ns | 2.04 GB/s | 0 B, 0 allocs | 6.84× (+584.0%) |
| `utf16.Decode-core` | chinese | 64.02 KiB / 32780 chars | **0.142 ns** | **14.11 GB/s** | 0 B, 0 allocs | 1.459 ns | 1.37 GB/s | 0 B, 0 allocs | 10.29× (+929.4%) |

## Reproduce

```text
$env:GOEXPERIMENT = 'simd'
D:\gosimd\.tools\go1.27rc1\bin\go.exe test -run=^$ -bench=^BenchmarkReport$ -benchmem -benchtime=1s -count=5 ./utf8
D:\gosimd\.tools\go1.27rc1\bin\go.exe test -run=^$ -bench=^BenchmarkReport$ -benchmem -benchtime=1s -count=5 ./utf16
```
