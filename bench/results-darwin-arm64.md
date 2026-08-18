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
| Go | `go1.27rc3-X:simd` with `GOEXPERIMENT=simd` |
| Git revision | `8be72ec75e80+dirty` |
| Generated (UTC) | `2026-08-18T14:10:27Z` |
| Sampling | median of 5 samples, `-benchtime=1s` |

## Workloads

Every row uses an approximately 64 KiB input working set. `ascii-only` is English ASCII; `mixed` combines English, Russian, Chinese, and emoji; `russian` and `chinese` contain only their named scripts. Repetition ends only at a valid encoding boundary.

For UTF-8 Valid, RuneCount, and Decode, throughput counts UTF-8 input bytes; UTF-8 Encode counts its 4-byte Go `rune` input. UTF-16 Encode counts the 4-byte Go `rune` input, and Decode counts the 2-byte UTF-16 input, matching the package benchmarks. A character means one decoded Unicode code point. `-full` calls the public API and includes output allocation; `-core` reuses caller-owned output. UTF-8 SIMD core rows measure only the encoder or decoder after their planning pass.

## UTF-8

| API | Scenario | Input | gosimd time/char | gosimd throughput | gosimd allocation | stdlib time/char | stdlib throughput | stdlib allocation | Speedup |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `utf8.Valid` | ascii-only | 64.03 KiB / 65565 chars | 0.011 ns | 88.10 GB/s | 0 B, 0 allocs | **0.009 ns** | **106.38 GB/s** | 0 B, 0 allocs | 0.83× (-17.2%) |
| `utf8.Valid` | mixed | 64.01 KiB / 43125 chars | **0.108 ns** | **14.13 GB/s** | 0 B, 0 allocs | 0.479 ns | 3.17 GB/s | 0 B, 0 allocs | 4.46× (+345.6%) |
| `utf8.Valid` | russian | 64.00 KiB / 36410 chars | **0.128 ns** | **14.09 GB/s** | 0 B, 0 allocs | 0.792 ns | 2.27 GB/s | 0 B, 0 allocs | 6.20× (+519.8%) |
| `utf8.Valid` | chinese | 64.00 KiB / 21846 chars | **0.216 ns** | **13.92 GB/s** | 0 B, 0 allocs | 0.926 ns | 3.24 GB/s | 0 B, 0 allocs | 4.30× (+329.8%) |
| `utf8.RuneCount` | ascii-only | 64.03 KiB / 65565 chars | **0.012 ns** | **83.87 GB/s** | 0 B, 0 allocs | 0.223 ns | 4.48 GB/s | 0 B, 0 allocs | 18.71× (+1771.4%) |
| `utf8.RuneCount` | mixed | 64.01 KiB / 43125 chars | **0.139 ns** | **10.96 GB/s** | 0 B, 0 allocs | 0.730 ns | 2.08 GB/s | 72.00 KiB, 1 allocs | 5.26× (+426.3%) |
| `utf8.RuneCount` | russian | 64.00 KiB / 36410 chars | **0.165 ns** | **10.93 GB/s** | 0 B, 0 allocs | 1.218 ns | 1.48 GB/s | 72.00 KiB, 1 allocs | 7.39× (+639.3%) |
| `utf8.RuneCount` | chinese | 64.00 KiB / 21846 chars | **0.276 ns** | **10.89 GB/s** | 0 B, 0 allocs | 1.554 ns | 1.93 GB/s | 72.00 KiB, 1 allocs | 5.64× (+464.0%) |
| `utf8.Encode-full` | ascii-only | 64.00 KiB / 16384 chars | **0.422 ns** | **9.49 GB/s** | 18.00 KiB, 1 allocs | 2.303 ns | 1.74 GB/s | 18.00 KiB, 1 allocs | 5.46× (+446.2%) |
| `utf8.Encode-full` | mixed | 64.00 KiB / 16384 chars | **1.418 ns** | **2.82 GB/s** | 26.62 KiB, 1 allocs | 2.578 ns | 1.55 GB/s | 26.62 KiB, 1 allocs | 1.82× (+81.8%) |
| `utf8.Encode-full` | russian | 64.00 KiB / 16384 chars | **1.834 ns** | **2.18 GB/s** | 32.00 KiB, 1 allocs | 2.558 ns | 1.56 GB/s | 32.00 KiB, 1 allocs | 1.40× (+39.5%) |
| `utf8.Encode-full` | chinese | 64.00 KiB / 16384 chars | **1.571 ns** | **2.55 GB/s** | 56.00 KiB, 1 allocs | 3.158 ns | 1.27 GB/s | 56.00 KiB, 1 allocs | 2.01× (+101.1%) |
| `utf8.Encode-core` | ascii-only | 64.00 KiB / 16384 chars | **0.040 ns** | **100.21 GB/s** | 0 B, 0 allocs | 0.446 ns | 8.96 GB/s | 0 B, 0 allocs | 11.18× (+1018.0%) |
| `utf8.Encode-core` | mixed | 64.00 KiB / 16384 chars | 0.999 ns | 4.01 GB/s | 0 B, 0 allocs | **0.840 ns** | **4.76 GB/s** | 0 B, 0 allocs | 0.84× (-15.9%) |
| `utf8.Encode-core` | russian | 64.00 KiB / 16384 chars | 1.440 ns | 2.78 GB/s | 0 B, 0 allocs | **1.207 ns** | **3.31 GB/s** | 0 B, 0 allocs | 0.84× (-16.1%) |
| `utf8.Encode-core` | chinese | 64.00 KiB / 16384 chars | **1.093 ns** | **3.66 GB/s** | 0 B, 0 allocs | 1.818 ns | 2.20 GB/s | 0 B, 0 allocs | 1.66× (+66.3%) |
| `utf8.Decode-full` | ascii-only | 64.03 KiB / 65565 chars | **0.222 ns** | **4.51 GB/s** | 264.00 KiB, 1 allocs | 0.834 ns | 1.20 GB/s | 264.00 KiB, 1 allocs | 3.76× (+275.9%) |
| `utf8.Decode-full` | mixed | 64.01 KiB / 43125 chars | **0.982 ns** | **1.55 GB/s** | 176.00 KiB, 1 allocs | 1.628 ns | 933.81 MB/s | 176.00 KiB, 1 allocs | 1.66× (+65.7%) |
| `utf8.Decode-full` | russian | 64.00 KiB / 36410 chars | **1.121 ns** | **1.61 GB/s** | 144.00 KiB, 1 allocs | 2.466 ns | 729.85 MB/s | 144.00 KiB, 1 allocs | 2.20× (+120.0%) |
| `utf8.Decode-full` | chinese | 64.00 KiB / 21846 chars | **1.159 ns** | **2.59 GB/s** | 88.00 KiB, 1 allocs | 2.941 ns | 1.02 GB/s | 88.00 KiB, 1 allocs | 2.54× (+153.7%) |
| `utf8.Decode-core` | ascii-only | 64.03 KiB / 65565 chars | **0.053 ns** | **18.78 GB/s** | 0 B, 0 allocs | 0.585 ns | 1.71 GB/s | 0 B, 0 allocs | 10.98× (+998.2%) |
| `utf8.Decode-core` | mixed | 64.01 KiB / 43125 chars | **0.688 ns** | **2.21 GB/s** | 0 B, 0 allocs | 1.229 ns | 1.24 GB/s | 0 B, 0 allocs | 1.79× (+78.7%) |
| `utf8.Decode-core` | russian | 64.00 KiB / 36410 chars | **0.833 ns** | **2.16 GB/s** | 0 B, 0 allocs | 1.745 ns | 1.03 GB/s | 0 B, 0 allocs | 2.10× (+109.5%) |
| `utf8.Decode-core` | chinese | 64.00 KiB / 21846 chars | **0.737 ns** | **4.07 GB/s** | 0 B, 0 allocs | 1.959 ns | 1.53 GB/s | 0 B, 0 allocs | 2.66× (+166.0%) |

## UTF-16

| API | Scenario | Input | gosimd time/char | gosimd throughput | gosimd allocation | stdlib time/char | stdlib throughput | stdlib allocation | Speedup |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `utf16.Encode-full` | ascii-only | 64.00 KiB / 16384 chars | **0.299 ns** | **13.37 GB/s** | 32.00 KiB, 1 allocs | 0.778 ns | 5.14 GB/s | 32.00 KiB, 1 allocs | 2.60× (+159.9%) |
| `utf16.Encode-full` | mixed | 64.00 KiB / 16384 chars | **0.533 ns** | **7.51 GB/s** | 40.00 KiB, 1 allocs | 1.154 ns | 3.46 GB/s | 40.00 KiB, 1 allocs | 2.17× (+116.7%) |
| `utf16.Encode-full` | russian | 64.00 KiB / 16384 chars | **0.294 ns** | **13.59 GB/s** | 32.00 KiB, 1 allocs | 0.777 ns | 5.15 GB/s | 32.00 KiB, 1 allocs | 2.64× (+164.0%) |
| `utf16.Encode-full` | chinese | 64.00 KiB / 16384 chars | **0.294 ns** | **13.61 GB/s** | 32.00 KiB, 1 allocs | 0.811 ns | 4.93 GB/s | 32.00 KiB, 1 allocs | 2.76× (+176.1%) |
| `utf16.Encode-core` | ascii-only | 64.00 KiB / 16384 chars | **0.212 ns** | **18.86 GB/s** | 0 B, 0 allocs | 0.704 ns | 5.68 GB/s | 0 B, 0 allocs | 3.32× (+231.9%) |
| `utf16.Encode-core` | mixed | 64.00 KiB / 16384 chars | **0.428 ns** | **9.35 GB/s** | 0 B, 0 allocs | 0.779 ns | 5.13 GB/s | 0 B, 0 allocs | 1.82× (+82.2%) |
| `utf16.Encode-core` | russian | 64.00 KiB / 16384 chars | **0.212 ns** | **18.86 GB/s** | 0 B, 0 allocs | 0.696 ns | 5.75 GB/s | 0 B, 0 allocs | 3.28× (+228.3%) |
| `utf16.Encode-core` | chinese | 64.00 KiB / 16384 chars | **0.209 ns** | **19.11 GB/s** | 0 B, 0 allocs | 0.734 ns | 5.45 GB/s | 0 B, 0 allocs | 3.51× (+250.6%) |
| `utf16.Decode-full` | ascii-only | 64.07 KiB / 32805 chars | **0.248 ns** | **8.06 GB/s** | 136.00 KiB, 1 allocs | 1.724 ns | 1.16 GB/s | 657.63 KiB, 17 allocs | 6.95× (+594.8%) |
| `utf16.Decode-full` | mixed | 64.04 KiB / 31525 chars | **0.499 ns** | **4.17 GB/s** | 136.00 KiB, 1 allocs | 1.489 ns | 1.40 GB/s | 489.63 KiB, 16 allocs | 2.98× (+198.4%) |
| `utf16.Decode-full` | russian | 64.02 KiB / 32780 chars | **0.241 ns** | **8.31 GB/s** | 136.00 KiB, 1 allocs | 1.743 ns | 1.15 GB/s | 657.63 KiB, 17 allocs | 7.24× (+623.9%) |
| `utf16.Decode-full` | chinese | 64.02 KiB / 32780 chars | **0.242 ns** | **8.27 GB/s** | 136.00 KiB, 1 allocs | 2.128 ns | 939.79 MB/s | 657.63 KiB, 17 allocs | 8.80× (+780.5%) |
| `utf16.Decode-core` | ascii-only | 64.07 KiB / 32805 chars | **0.057 ns** | **34.81 GB/s** | 0 B, 0 allocs | 0.462 ns | 4.32 GB/s | 0 B, 0 allocs | 8.05× (+704.8%) |
| `utf16.Decode-core` | mixed | 64.04 KiB / 31525 chars | **0.326 ns** | **6.37 GB/s** | 0 B, 0 allocs | 0.603 ns | 3.45 GB/s | 0 B, 0 allocs | 1.85× (+84.9%) |
| `utf16.Decode-core` | russian | 64.02 KiB / 32780 chars | **0.057 ns** | **34.82 GB/s** | 0 B, 0 allocs | 0.461 ns | 4.34 GB/s | 0 B, 0 allocs | 8.03× (+702.9%) |
| `utf16.Decode-core` | chinese | 64.02 KiB / 32780 chars | **0.057 ns** | **34.82 GB/s** | 0 B, 0 allocs | 0.706 ns | 2.83 GB/s | 0 B, 0 allocs | 12.29× (+1128.9%) |

## Reproduce

```text
GOEXPERIMENT=simd ../.tools/go1.27rc3/bin/go test -run=^$ -bench=^BenchmarkReport$ -benchmem -benchtime=1s -count=5 ./utf8
GOEXPERIMENT=simd ../.tools/go1.27rc3/bin/go test -run=^$ -bench=^BenchmarkReport$ -benchmem -benchtime=1s -count=5 ./utf16
```
