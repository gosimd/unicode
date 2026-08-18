# Unicode performance on Intel Xeon Processor (Skylake, IBRS, no TSX)

This report records the expected performance level for this machine. Results are medians; higher throughput and a larger speedup are better, while lower time per character is better.

## Environment

| Parameter | Value |
|---|---|
| CPU | Intel Xeon Processor (Skylake, IBRS, no TSX) |
| Frequency | 2.10 GHz (maximum observed) |
| Active SIMD backend | AVX-512 |
| Logical CPUs | 2 |
| Platform | `linux/amd64` |
| Go | `go1.27rc3-X:simd` with `GOEXPERIMENT=simd` |
| Git revision | `a4358d794d89+dirty` |
| Generated (UTC) | `2026-08-18T12:00:45Z` |
| Sampling | median of 5 samples, `-benchtime=1s` |

## Workloads

Every row uses an approximately 64 KiB input working set. `ascii-only` is English ASCII; `mixed` combines English, Russian, Chinese, and emoji; `russian` and `chinese` contain only their named scripts. Repetition ends only at a valid encoding boundary.

For UTF-8 Valid, RuneCount, and Decode, throughput counts UTF-8 input bytes; UTF-8 Encode counts its 4-byte Go `rune` input. UTF-16 Encode counts the 4-byte Go `rune` input, and Decode counts the 2-byte UTF-16 input, matching the package benchmarks. A character means one decoded Unicode code point. `-full` calls the public API and includes output allocation; `-core` reuses caller-owned output. UTF-8 SIMD core rows measure only the encoder or decoder after their planning pass.

## UTF-8

| API | Scenario | Input | gosimd time/char | gosimd throughput | gosimd allocation | stdlib time/char | stdlib throughput | stdlib allocation | Speedup |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `utf8.Valid` | ascii-only | 64.03 KiB / 65565 chars | **0.021 ns** | **48.32 GB/s** | 0 B, 0 allocs | 0.046 ns | 21.95 GB/s | 0 B, 0 allocs | 2.20× (+120.1%) |
| `utf8.Valid` | mixed | 64.01 KiB / 43125 chars | **0.198 ns** | **7.67 GB/s** | 0 B, 0 allocs | 2.175 ns | 698.75 MB/s | 0 B, 0 allocs | 10.97× (+997.1%) |
| `utf8.Valid` | russian | 64.00 KiB / 36410 chars | **0.236 ns** | **7.64 GB/s** | 0 B, 0 allocs | 3.328 ns | 540.84 MB/s | 0 B, 0 allocs | 14.13× (+1312.7%) |
| `utf8.Valid` | chinese | 64.00 KiB / 21846 chars | **0.372 ns** | **8.07 GB/s** | 0 B, 0 allocs | 4.345 ns | 690.38 MB/s | 0 B, 0 allocs | 11.68× (+1068.2%) |
| `utf8.RuneCount` | ascii-only | 64.03 KiB / 65565 chars | **0.021 ns** | **48.28 GB/s** | 0 B, 0 allocs | 0.897 ns | 1.11 GB/s | 0 B, 0 allocs | 43.31× (+4230.7%) |
| `utf8.RuneCount` | mixed | 64.01 KiB / 43125 chars | **0.219 ns** | **6.94 GB/s** | 0 B, 0 allocs | 3.741 ns | 406.29 MB/s | 72.00 KiB, 1 allocs | 17.07× (+1607.3%) |
| `utf8.RuneCount` | russian | 64.00 KiB / 36410 chars | **0.282 ns** | **6.39 GB/s** | 0 B, 0 allocs | 5.256 ns | 342.50 MB/s | 72.00 KiB, 1 allocs | 18.66× (+1765.6%) |
| `utf8.RuneCount` | chinese | 64.00 KiB / 21846 chars | **0.458 ns** | **6.55 GB/s** | 0 B, 0 allocs | 8.114 ns | 369.74 MB/s | 72.00 KiB, 1 allocs | 17.72× (+1672.4%) |
| `utf8.Encode-full` | ascii-only | 64.00 KiB / 16384 chars | **0.638 ns** | **6.27 GB/s** | 18.00 KiB, 1 allocs | 9.562 ns | 418.31 MB/s | 18.00 KiB, 1 allocs | 15.00× (+1399.5%) |
| `utf8.Encode-full` | mixed | 64.00 KiB / 16384 chars | **3.291 ns** | **1.22 GB/s** | 26.62 KiB, 1 allocs | 10.963 ns | 364.88 MB/s | 26.62 KiB, 1 allocs | 3.33× (+233.1%) |
| `utf8.Encode-full` | russian | 64.00 KiB / 16384 chars | **3.691 ns** | **1.08 GB/s** | 32.00 KiB, 1 allocs | 11.704 ns | 341.77 MB/s | 32.00 KiB, 1 allocs | 3.17× (+217.1%) |
| `utf8.Encode-full` | chinese | 64.00 KiB / 16384 chars | **3.168 ns** | **1.26 GB/s** | 56.00 KiB, 1 allocs | 15.694 ns | 254.87 MB/s | 56.00 KiB, 1 allocs | 4.95× (+395.4%) |
| `utf8.Encode-core` | ascii-only | 64.00 KiB / 16384 chars | **0.111 ns** | **36.13 GB/s** | 0 B, 0 allocs | 1.669 ns | 2.40 GB/s | 0 B, 0 allocs | 15.08× (+1407.6%) |
| `utf8.Encode-core` | mixed | 64.00 KiB / 16384 chars | **1.702 ns** | **2.35 GB/s** | 0 B, 0 allocs | 3.513 ns | 1.14 GB/s | 0 B, 0 allocs | 2.06× (+106.4%) |
| `utf8.Encode-core` | russian | 64.00 KiB / 16384 chars | **1.740 ns** | **2.30 GB/s** | 0 B, 0 allocs | 5.352 ns | 747.33 MB/s | 0 B, 0 allocs | 3.08× (+207.7%) |
| `utf8.Encode-core` | chinese | 64.00 KiB / 16384 chars | **0.800 ns** | **5.00 GB/s** | 0 B, 0 allocs | 7.394 ns | 540.96 MB/s | 0 B, 0 allocs | 9.24× (+824.4%) |
| `utf8.Decode-full` | ascii-only | 64.03 KiB / 65565 chars | **1.149 ns** | **870.64 MB/s** | 264.00 KiB, 1 allocs | 3.508 ns | 285.09 MB/s | 264.00 KiB, 1 allocs | 3.05× (+205.4%) |
| `utf8.Decode-full` | mixed | 64.01 KiB / 43125 chars | **2.764 ns** | **549.89 MB/s** | 176.00 KiB, 1 allocs | 7.935 ns | 191.57 MB/s | 176.00 KiB, 1 allocs | 2.87× (+187.1%) |
| `utf8.Decode-full` | russian | 64.00 KiB / 36410 chars | **3.432 ns** | **524.55 MB/s** | 144.00 KiB, 1 allocs | 10.582 ns | 170.10 MB/s | 144.00 KiB, 1 allocs | 3.08× (+208.4%) |
| `utf8.Decode-full` | chinese | 64.00 KiB / 21846 chars | **2.590 ns** | **1.16 GB/s** | 88.00 KiB, 1 allocs | 16.381 ns | 183.14 MB/s | 88.00 KiB, 1 allocs | 6.32× (+532.5%) |
| `utf8.Decode-core` | ascii-only | 64.03 KiB / 65565 chars | **0.140 ns** | **7.13 GB/s** | 0 B, 0 allocs | 2.454 ns | 407.48 MB/s | 0 B, 0 allocs | 17.49× (+1648.9%) |
| `utf8.Decode-core` | mixed | 64.01 KiB / 43125 chars | **1.159 ns** | **1.31 GB/s** | 0 B, 0 allocs | 4.643 ns | 327.37 MB/s | 0 B, 0 allocs | 4.01× (+300.7%) |
| `utf8.Decode-core` | russian | 64.00 KiB / 36410 chars | **1.362 ns** | **1.32 GB/s** | 0 B, 0 allocs | 7.289 ns | 246.96 MB/s | 0 B, 0 allocs | 5.35× (+435.0%) |
| `utf8.Decode-core` | chinese | 64.00 KiB / 21846 chars | **0.576 ns** | **5.21 GB/s** | 0 B, 0 allocs | 8.200 ns | 365.87 MB/s | 0 B, 0 allocs | 14.23× (+1323.5%) |

## UTF-16

| API | Scenario | Input | gosimd time/char | gosimd throughput | gosimd allocation | stdlib time/char | stdlib throughput | stdlib allocation | Speedup |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `utf16.Encode-full` | ascii-only | 64.00 KiB / 16384 chars | **1.271 ns** | **3.15 GB/s** | 32.00 KiB, 1 allocs | 5.190 ns | 770.77 MB/s | 32.00 KiB, 1 allocs | 4.08× (+308.4%) |
| `utf16.Encode-full` | mixed | 64.00 KiB / 16384 chars | **2.909 ns** | **1.37 GB/s** | 40.00 KiB, 1 allocs | 5.535 ns | 722.64 MB/s | 40.00 KiB, 1 allocs | 1.90× (+90.3%) |
| `utf16.Encode-full` | russian | 64.00 KiB / 16384 chars | **1.475 ns** | **2.71 GB/s** | 32.00 KiB, 1 allocs | 5.663 ns | 706.33 MB/s | 32.00 KiB, 1 allocs | 3.84× (+283.9%) |
| `utf16.Encode-full` | chinese | 64.00 KiB / 16384 chars | **1.461 ns** | **2.74 GB/s** | 32.00 KiB, 1 allocs | 5.816 ns | 687.79 MB/s | 32.00 KiB, 1 allocs | 3.98× (+298.1%) |
| `utf16.Encode-core` | ascii-only | 64.00 KiB / 16384 chars | **0.370 ns** | **10.80 GB/s** | 0 B, 0 allocs | 3.285 ns | 1.22 GB/s | 0 B, 0 allocs | 8.87× (+786.9%) |
| `utf16.Encode-core` | mixed | 64.00 KiB / 16384 chars | **1.628 ns** | **2.46 GB/s** | 0 B, 0 allocs | 3.262 ns | 1.23 GB/s | 0 B, 0 allocs | 2.00× (+100.4%) |
| `utf16.Encode-core` | russian | 64.00 KiB / 16384 chars | **0.417 ns** | **9.59 GB/s** | 0 B, 0 allocs | 3.333 ns | 1.20 GB/s | 0 B, 0 allocs | 7.99× (+699.3%) |
| `utf16.Encode-core` | chinese | 64.00 KiB / 16384 chars | **0.405 ns** | **9.87 GB/s** | 0 B, 0 allocs | 3.120 ns | 1.28 GB/s | 0 B, 0 allocs | 7.70× (+669.5%) |
| `utf16.Decode-full` | ascii-only | 64.07 KiB / 32805 chars | **2.070 ns** | **966.36 MB/s** | 136.00 KiB, 1 allocs | 12.194 ns | 164.01 MB/s | 657.62 KiB, 17 allocs | 5.89× (+489.2%) |
| `utf16.Decode-full` | mixed | 64.04 KiB / 31525 chars | **4.048 ns** | **513.90 MB/s** | 136.00 KiB, 1 allocs | 11.325 ns | 183.67 MB/s | 489.62 KiB, 16 allocs | 2.80× (+179.8%) |
| `utf16.Decode-full` | russian | 64.02 KiB / 32780 chars | **2.298 ns** | **870.18 MB/s** | 136.00 KiB, 1 allocs | 17.013 ns | 117.56 MB/s | 657.62 KiB, 17 allocs | 7.40× (+640.2%) |
| `utf16.Decode-full` | chinese | 64.02 KiB / 32780 chars | **2.368 ns** | **844.74 MB/s** | 136.00 KiB, 1 allocs | 18.139 ns | 110.26 MB/s | 657.62 KiB, 17 allocs | 7.66× (+666.1%) |
| `utf16.Decode-core` | ascii-only | 64.07 KiB / 32805 chars | **0.161 ns** | **12.44 GB/s** | 0 B, 0 allocs | 2.074 ns | 964.16 MB/s | 0 B, 0 allocs | 12.91× (+1190.5%) |
| `utf16.Decode-core` | mixed | 64.04 KiB / 31525 chars | **1.736 ns** | **1.20 GB/s** | 0 B, 0 allocs | 2.311 ns | 899.89 MB/s | 0 B, 0 allocs | 1.33× (+33.2%) |
| `utf16.Decode-core` | russian | 64.02 KiB / 32780 chars | **0.154 ns** | **12.99 GB/s** | 0 B, 0 allocs | 2.056 ns | 972.61 MB/s | 0 B, 0 allocs | 13.35× (+1235.3%) |
| `utf16.Decode-core` | chinese | 64.02 KiB / 32780 chars | **0.158 ns** | **12.66 GB/s** | 0 B, 0 allocs | 2.458 ns | 813.55 MB/s | 0 B, 0 allocs | 15.56× (+1456.3%) |

## Reproduce

```text
GOEXPERIMENT=simd /root/.local/go1.27rc3/bin/go test -run=^$ -bench=^BenchmarkReport$ -benchmem -benchtime=1s -count=5 ./utf8
GOEXPERIMENT=simd /root/.local/go1.27rc3/bin/go test -run=^$ -bench=^BenchmarkReport$ -benchmem -benchtime=1s -count=5 ./utf16
```
