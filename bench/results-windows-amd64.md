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
| Go | `go1.27rc3-X:simd` with `GOEXPERIMENT=simd` |
| Git revision | `cfd417c020af` |
| Generated (UTC) | `2026-08-18T14:09:55Z` |
| Sampling | median of 5 samples, `-benchtime=1s` |

## Workloads

Every row uses an approximately 64 KiB input working set. `ascii-only` is English ASCII; `mixed` combines English, Russian, Chinese, and emoji; `russian` and `chinese` contain only their named scripts. Repetition ends only at a valid encoding boundary.

For UTF-8 Valid, RuneCount, and Decode, throughput counts UTF-8 input bytes; UTF-8 Encode counts its 4-byte Go `rune` input. UTF-16 Encode counts the 4-byte Go `rune` input, and Decode counts the 2-byte UTF-16 input, matching the package benchmarks. A character means one decoded Unicode code point. `-full` calls the public API and includes output allocation; `-core` reuses caller-owned output. UTF-8 SIMD core rows measure only the encoder or decoder after their planning pass.

## UTF-8

| API | Scenario | Input | gosimd time/char | gosimd throughput | gosimd allocation | stdlib time/char | stdlib throughput | stdlib allocation | Speedup |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `utf8.Valid` | ascii-only | 64.03 KiB / 65565 chars | **0.013 ns** | **78.27 GB/s** | 0 B, 0 allocs | 0.034 ns | 29.08 GB/s | 0 B, 0 allocs | 2.69× (+169.2%) |
| `utf8.Valid` | mixed | 64.01 KiB / 43125 chars | **0.168 ns** | **9.04 GB/s** | 0 B, 0 allocs | 1.498 ns | 1.01 GB/s | 0 B, 0 allocs | 8.91× (+790.6%) |
| `utf8.Valid` | russian | 64.00 KiB / 36410 chars | **0.201 ns** | **8.96 GB/s** | 0 B, 0 allocs | 2.347 ns | 766.97 MB/s | 0 B, 0 allocs | 11.68× (+1068.1%) |
| `utf8.Valid` | chinese | 64.00 KiB / 21846 chars | **0.319 ns** | **9.40 GB/s** | 0 B, 0 allocs | 3.108 ns | 965.28 MB/s | 0 B, 0 allocs | 9.74× (+874.0%) |
| `utf8.RuneCount` | ascii-only | 64.03 KiB / 65565 chars | **0.013 ns** | **74.62 GB/s** | 0 B, 0 allocs | 0.329 ns | 3.04 GB/s | 0 B, 0 allocs | 24.53× (+2353.1%) |
| `utf8.RuneCount` | mixed | 64.01 KiB / 43125 chars | **0.185 ns** | **8.23 GB/s** | 0 B, 0 allocs | 2.050 ns | 741.36 MB/s | 72.00 KiB, 1 allocs | 11.10× (+1010.4%) |
| `utf8.RuneCount` | russian | 64.00 KiB / 36410 chars | **0.222 ns** | **8.12 GB/s** | 0 B, 0 allocs | 3.395 ns | 530.18 MB/s | 72.00 KiB, 1 allocs | 15.31× (+1431.2%) |
| `utf8.RuneCount` | chinese | 64.00 KiB / 21846 chars | **0.351 ns** | **8.54 GB/s** | 0 B, 0 allocs | 4.631 ns | 647.74 MB/s | 72.00 KiB, 1 allocs | 13.19× (+1219.0%) |
| `utf8.Encode-full` | ascii-only | 64.00 KiB / 16384 chars | **0.441 ns** | **9.07 GB/s** | 18.00 KiB, 1 allocs | 5.923 ns | 675.32 MB/s | 18.00 KiB, 1 allocs | 13.43× (+1243.4%) |
| `utf8.Encode-full` | mixed | 64.00 KiB / 16384 chars | **1.696 ns** | **2.36 GB/s** | 26.62 KiB, 1 allocs | 6.667 ns | 599.99 MB/s | 26.62 KiB, 1 allocs | 3.93× (+293.1%) |
| `utf8.Encode-full` | russian | 64.00 KiB / 16384 chars | **1.773 ns** | **2.26 GB/s** | 32.00 KiB, 1 allocs | 8.192 ns | 488.26 MB/s | 32.00 KiB, 1 allocs | 4.62× (+362.1%) |
| `utf8.Encode-full` | chinese | 64.00 KiB / 16384 chars | **1.797 ns** | **2.23 GB/s** | 56.00 KiB, 1 allocs | 9.173 ns | 436.08 MB/s | 56.00 KiB, 1 allocs | 5.11× (+410.6%) |
| `utf8.Encode-core` | ascii-only | 64.00 KiB / 16384 chars | **0.166 ns** | **24.09 GB/s** | 0 B, 0 allocs | 0.984 ns | 4.06 GB/s | 0 B, 0 allocs | 5.93× (+492.8%) |
| `utf8.Encode-core` | mixed | 64.00 KiB / 16384 chars | **1.072 ns** | **3.73 GB/s** | 0 B, 0 allocs | 2.236 ns | 1.79 GB/s | 0 B, 0 allocs | 2.08× (+108.5%) |
| `utf8.Encode-core` | russian | 64.00 KiB / 16384 chars | **1.090 ns** | **3.67 GB/s** | 0 B, 0 allocs | 3.670 ns | 1.09 GB/s | 0 B, 0 allocs | 3.37× (+236.8%) |
| `utf8.Encode-core` | chinese | 64.00 KiB / 16384 chars | **0.757 ns** | **5.29 GB/s** | 0 B, 0 allocs | 4.625 ns | 864.78 MB/s | 0 B, 0 allocs | 6.11× (+511.2%) |
| `utf8.Decode-full` | ascii-only | 64.03 KiB / 65565 chars | **0.612 ns** | **1.63 GB/s** | 264.00 KiB, 1 allocs | 2.158 ns | 463.43 MB/s | 264.00 KiB, 1 allocs | 3.52× (+252.5%) |
| `utf8.Decode-full` | mixed | 64.01 KiB / 43125 chars | **2.124 ns** | **715.56 MB/s** | 176.00 KiB, 1 allocs | 4.264 ns | 356.47 MB/s | 176.00 KiB, 1 allocs | 2.01× (+100.7%) |
| `utf8.Decode-full` | russian | 64.00 KiB / 36410 chars | **2.288 ns** | **786.70 MB/s** | 144.00 KiB, 1 allocs | 7.164 ns | 251.25 MB/s | 144.00 KiB, 1 allocs | 3.13× (+213.1%) |
| `utf8.Decode-full` | chinese | 64.00 KiB / 21846 chars | **2.063 ns** | **1.45 GB/s** | 88.00 KiB, 1 allocs | 9.157 ns | 327.64 MB/s | 88.00 KiB, 1 allocs | 4.44× (+343.9%) |
| `utf8.Decode-core` | ascii-only | 64.03 KiB / 65565 chars | **0.132 ns** | **7.59 GB/s** | 0 B, 0 allocs | 1.671 ns | 598.29 MB/s | 0 B, 0 allocs | 12.68× (+1168.4%) |
| `utf8.Decode-core` | mixed | 64.01 KiB / 43125 chars | **1.497 ns** | **1.02 GB/s** | 0 B, 0 allocs | 3.175 ns | 478.77 MB/s | 0 B, 0 allocs | 2.12× (+112.1%) |
| `utf8.Decode-core` | russian | 64.00 KiB / 36410 chars | **1.652 ns** | **1.09 GB/s** | 0 B, 0 allocs | 5.586 ns | 322.24 MB/s | 0 B, 0 allocs | 3.38× (+238.2%) |
| `utf8.Decode-core` | chinese | 64.00 KiB / 21846 chars | **0.911 ns** | **3.29 GB/s** | 0 B, 0 allocs | 5.957 ns | 503.59 MB/s | 0 B, 0 allocs | 6.54× (+554.0%) |

## UTF-16

| API | Scenario | Input | gosimd time/char | gosimd throughput | gosimd allocation | stdlib time/char | stdlib throughput | stdlib allocation | Speedup |
|---|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `utf16.Encode-full` | ascii-only | 64.00 KiB / 16384 chars | **0.577 ns** | **6.93 GB/s** | 32.00 KiB, 1 allocs | 2.442 ns | 1.64 GB/s | 32.00 KiB, 1 allocs | 4.23× (+323.3%) |
| `utf16.Encode-full` | mixed | 64.00 KiB / 16384 chars | **1.159 ns** | **3.45 GB/s** | 40.00 KiB, 1 allocs | 2.649 ns | 1.51 GB/s | 40.00 KiB, 1 allocs | 2.29× (+128.6%) |
| `utf16.Encode-full` | russian | 64.00 KiB / 16384 chars | **0.610 ns** | **6.56 GB/s** | 32.00 KiB, 1 allocs | 2.494 ns | 1.60 GB/s | 32.00 KiB, 1 allocs | 4.09× (+308.9%) |
| `utf16.Encode-full` | chinese | 64.00 KiB / 16384 chars | **0.745 ns** | **5.37 GB/s** | 32.00 KiB, 1 allocs | 2.524 ns | 1.58 GB/s | 32.00 KiB, 1 allocs | 3.39× (+239.0%) |
| `utf16.Encode-core` | ascii-only | 64.00 KiB / 16384 chars | **0.150 ns** | **26.65 GB/s** | 0 B, 0 allocs | 2.025 ns | 1.97 GB/s | 0 B, 0 allocs | 13.50× (+1249.5%) |
| `utf16.Encode-core` | mixed | 64.00 KiB / 16384 chars | **0.664 ns** | **6.03 GB/s** | 0 B, 0 allocs | 2.146 ns | 1.86 GB/s | 0 B, 0 allocs | 3.23× (+223.4%) |
| `utf16.Encode-core` | russian | 64.00 KiB / 16384 chars | **0.150 ns** | **26.63 GB/s** | 0 B, 0 allocs | 2.024 ns | 1.98 GB/s | 0 B, 0 allocs | 13.48× (+1247.6%) |
| `utf16.Encode-core` | chinese | 64.00 KiB / 16384 chars | **0.348 ns** | **11.51 GB/s** | 0 B, 0 allocs | 2.115 ns | 1.89 GB/s | 0 B, 0 allocs | 6.08× (+508.4%) |
| `utf16.Decode-full` | ascii-only | 64.07 KiB / 32805 chars | **0.886 ns** | **2.26 GB/s** | 136.00 KiB, 1 allocs | 6.215 ns | 321.79 MB/s | 657.63 KiB, 17 allocs | 7.01× (+601.2%) |
| `utf16.Decode-full` | mixed | 64.04 KiB / 31525 chars | **1.153 ns** | **1.80 GB/s** | 136.00 KiB, 1 allocs | 5.411 ns | 384.40 MB/s | 489.63 KiB, 16 allocs | 4.69× (+369.1%) |
| `utf16.Decode-full` | russian | 64.02 KiB / 32780 chars | **0.883 ns** | **2.26 GB/s** | 136.00 KiB, 1 allocs | 6.191 ns | 323.05 MB/s | 657.63 KiB, 17 allocs | 7.01× (+600.9%) |
| `utf16.Decode-full` | chinese | 64.02 KiB / 32780 chars | **0.905 ns** | **2.21 GB/s** | 136.00 KiB, 1 allocs | 6.949 ns | 287.83 MB/s | 657.63 KiB, 17 allocs | 7.68× (+667.6%) |
| `utf16.Decode-core` | ascii-only | 64.07 KiB / 32805 chars | **0.142 ns** | **14.06 GB/s** | 0 B, 0 allocs | 1.315 ns | 1.52 GB/s | 0 B, 0 allocs | 9.25× (+824.6%) |
| `utf16.Decode-core` | mixed | 64.04 KiB / 31525 chars | **0.376 ns** | **5.53 GB/s** | 0 B, 0 allocs | 1.521 ns | 1.37 GB/s | 0 B, 0 allocs | 4.04× (+304.3%) |
| `utf16.Decode-core` | russian | 64.02 KiB / 32780 chars | **0.144 ns** | **13.89 GB/s** | 0 B, 0 allocs | 1.327 ns | 1.51 GB/s | 0 B, 0 allocs | 9.21× (+821.4%) |
| `utf16.Decode-core` | chinese | 64.02 KiB / 32780 chars | **0.141 ns** | **14.14 GB/s** | 0 B, 0 allocs | 1.819 ns | 1.10 GB/s | 0 B, 0 allocs | 12.86× (+1185.5%) |

## Reproduce

```text
$env:GOEXPERIMENT = 'simd'
C:\Program files\Go\bin\go.exe test -run=^$ -bench=^BenchmarkReport$ -benchmem -benchtime=1s -count=5 ./utf8
C:\Program files\Go\bin\go.exe test -run=^$ -bench=^BenchmarkReport$ -benchmem -benchtime=1s -count=5 ./utf16
```
