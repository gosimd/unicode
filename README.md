# utf
SIMD implementation for UTF-encodings

## Development

This repository is configured for Go 1.27rc1 via the local toolchain at
`../.tools/go1.27rc1`.

Common commands:

```sh
make build
make test
make bench
make profile
make profile-cpu
make profile-mem
```

In VS Code, install the recommended Go extension and use the provided tasks:

- `Go: build`
- `Go: test`
- `Go: benchmark`
- `Go: profile benchmark`
- `Go: pprof CPU`
- `Go: pprof memory`
