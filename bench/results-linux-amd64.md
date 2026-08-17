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
| Go | `go1.27rc1-X:simd` with `GOEXPERIMENT=simd` |
| Git revision | `d2051cb48ba1+dirty` |
| Generated (UTC) | `2026-08-17T19:15:32Z` |
| Sampling | median of 5 samples, `-benchtime=1s` |

## Workloads

Every row uses an approximately 64 KiB input working set. `ascii-only` is English ASCII; `mixed` combines English, Russian, Chinese, and emoji; `russian` and `chinese` contain only their named scripts. Repetition ends only at a valid encoding boundary.

For UTF-8, throughput counts UTF-8 input bytes. For UTF-16 Encode it counts the 4-byte Go `rune` input, and for Decode it counts the 2-byte UTF-16 input, matching the package benchmarks. A character means one decoded Unicode code point. `-full` calls the public API and includes output allocation; `-core` reuses caller-owned output but includes length/planning and conversion work.

## UTF-8

| API | Scenario | Input | gosimd time/char | gosimd throughput | gosimd allocation | stdlib time/char | stdlib throughput | stdlib allocation | Speedup |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `utf8.Valid` | ascii-only | 64.03 KiB / 65565 chars | **0.020 ns** | **50.36 GB/s** | 0 B, 0 allocs | 0.044 ns | 22.76 GB/s | 0 B, 0 allocs | 2.21× (+121.3%) |
| `utf8.Valid` | mixed | 64.01 KiB / 43125 chars | **0.184 ns** | **8.27 GB/s** | 0 B, 0 allocs | 2.102 ns | 722.95 MB/s | 0 B, 0 allocs | 11.43× (+1043.2%) |
| `utf8.Valid` | russian | 64.00 KiB / 36410 chars | **0.226 ns** | **7.96 GB/s** | 0 B, 0 allocs | 2.865 ns | 628.26 MB/s | 0 B, 0 allocs | 12.68× (+1167.5%) |
| `utf8.Valid` | chinese | 64.00 KiB / 21846 chars | **0.334 ns** | **8.97 GB/s** | 0 B, 0 allocs | 3.983 ns | 753.20 MB/s | 0 B, 0 allocs | 11.91× (+1090.8%) |
| `utf8.RuneCount` | ascii-only | 64.03 KiB / 65565 chars | **0.019 ns** | **51.30 GB/s** | 0 B, 0 allocs | 0.437 ns | 2.29 GB/s | 0 B, 0 allocs | 22.42× (+2142.4%) |
| `utf8.RuneCount` | mixed | 64.01 KiB / 43125 chars | **0.211 ns** | **7.20 GB/s** | 0 B, 0 allocs | 4.774 ns | 318.41 MB/s | 72.00 KiB, 1 allocs | 22.62× (+2162.3%) |
| `utf8.RuneCount` | russian | 64.00 KiB / 36410 chars | **0.235 ns** | **7.67 GB/s** | 0 B, 0 allocs | 7.020 ns | 256.40 MB/s | 72.00 KiB, 1 allocs | 29.90× (+2890.3%) |
| `utf8.RuneCount` | chinese | 64.00 KiB / 21846 chars | **0.383 ns** | **7.83 GB/s** | 0 B, 0 allocs | 11.618 ns | 258.23 MB/s | 72.00 KiB, 1 allocs | 30.33× (+2932.6%) |

## UTF-16

| API | Scenario | Input | gosimd time/char | gosimd throughput | gosimd allocation | stdlib time/char | stdlib throughput | stdlib allocation | Speedup |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `utf16.Encode-full` | ascii-only | 64.00 KiB / 16384 chars | **1.302 ns** | **3.07 GB/s** | 32.00 KiB, 1 allocs | 5.880 ns | 680.22 MB/s | 32.00 KiB, 1 allocs | 4.52× (+351.7%) |
| `utf16.Encode-full` | mixed | 64.00 KiB / 16384 chars | **3.532 ns** | **1.13 GB/s** | 40.00 KiB, 1 allocs | 6.810 ns | 587.40 MB/s | 40.00 KiB, 1 allocs | 1.93× (+92.8%) |
| `utf16.Encode-full` | russian | 64.00 KiB / 16384 chars | **1.422 ns** | **2.81 GB/s** | 32.00 KiB, 1 allocs | 5.458 ns | 732.87 MB/s | 32.00 KiB, 1 allocs | 3.84× (+283.8%) |
| `utf16.Encode-full` | chinese | 64.00 KiB / 16384 chars | **1.105 ns** | **3.62 GB/s** | 32.00 KiB, 1 allocs | 5.228 ns | 765.17 MB/s | 32.00 KiB, 1 allocs | 4.73× (+373.1%) |
| `utf16.Encode-core` | ascii-only | 64.00 KiB / 16384 chars | **0.338 ns** | **11.83 GB/s** | 0 B, 0 allocs | 3.512 ns | 1.14 GB/s | 0 B, 0 allocs | 10.39× (+938.6%) |
| `utf16.Encode-core` | mixed | 64.00 KiB / 16384 chars | **1.614 ns** | **2.48 GB/s** | 0 B, 0 allocs | 3.513 ns | 1.14 GB/s | 0 B, 0 allocs | 2.18× (+117.6%) |
| `utf16.Encode-core` | russian | 64.00 KiB / 16384 chars | **0.344 ns** | **11.62 GB/s** | 0 B, 0 allocs | 3.285 ns | 1.22 GB/s | 0 B, 0 allocs | 9.54× (+854.1%) |
| `utf16.Encode-core` | chinese | 64.00 KiB / 16384 chars | **0.355 ns** | **11.26 GB/s** | 0 B, 0 allocs | 3.494 ns | 1.14 GB/s | 0 B, 0 allocs | 9.83× (+883.4%) |
| `utf16.Decode-full` | ascii-only | 64.07 KiB / 32805 chars | **1.661 ns** | **1.20 GB/s** | 136.00 KiB, 1 allocs | 11.908 ns | 167.95 MB/s | 657.62 KiB, 17 allocs | 7.17× (+616.9%) |
| `utf16.Decode-full` | mixed | 64.04 KiB / 31525 chars | **3.606 ns** | **576.75 MB/s** | 136.00 KiB, 1 allocs | 9.712 ns | 214.18 MB/s | 489.62 KiB, 16 allocs | 2.69× (+169.3%) |
| `utf16.Decode-full` | russian | 64.02 KiB / 32780 chars | **1.744 ns** | **1.15 GB/s** | 136.00 KiB, 1 allocs | 14.024 ns | 142.61 MB/s | 657.62 KiB, 17 allocs | 8.04× (+703.9%) |
| `utf16.Decode-full` | chinese | 64.02 KiB / 32780 chars | **1.517 ns** | **1.32 GB/s** | 136.00 KiB, 1 allocs | 12.639 ns | 158.24 MB/s | 657.62 KiB, 17 allocs | 8.33× (+732.9%) |
| `utf16.Decode-core` | ascii-only | 64.07 KiB / 32805 chars | **0.140 ns** | **14.33 GB/s** | 0 B, 0 allocs | 1.413 ns | 1.42 GB/s | 0 B, 0 allocs | 10.12× (+912.3%) |
| `utf16.Decode-core` | mixed | 64.04 KiB / 31525 chars | **1.498 ns** | **1.39 GB/s** | 0 B, 0 allocs | 1.619 ns | 1.28 GB/s | 0 B, 0 allocs | 1.08× (+8.1%) |
| `utf16.Decode-core` | russian | 64.02 KiB / 32780 chars | **0.140 ns** | **14.25 GB/s** | 0 B, 0 allocs | 1.418 ns | 1.41 GB/s | 0 B, 0 allocs | 10.10× (+910.4%) |
| `utf16.Decode-core` | chinese | 64.02 KiB / 32780 chars | **0.141 ns** | **14.22 GB/s** | 0 B, 0 allocs | 2.049 ns | 976.02 MB/s | 0 B, 0 allocs | 14.57× (+1357.1%) |

## Reproduce

```text
GOEXPERIMENT=simd /root/.local/go1.27rc1/bin/go test -run=^$ -bench=^BenchmarkReport$ -benchmem -benchtime=1s -count=5 ./utf8
GOEXPERIMENT=simd /root/.local/go1.27rc1/bin/go test -run=^$ -bench=^BenchmarkReport$ -benchmem -benchtime=1s -count=5 ./utf16
```
