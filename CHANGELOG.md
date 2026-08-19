# Changelog

All notable changes to this project are documented in this file.

## [0.1.0] - 2026-08-19

### Added

- UTF-8 validation and rune-counting APIs with semantics matching
  `unicode/utf8`.
- Whole-buffer UTF-8 `Encode` and `Decode` extensions, equivalent to Go's
  `string([]rune)` and `[]rune(string)` conversions.
- UTF-16 `Encode` and `Decode` APIs compatible with `unicode/utf16`.
- SIMD implementations for supported ARM64 and AMD64 builds, with portable
  fallbacks for other configurations.
- API, implementation, benchmark, and compatibility documentation.

[0.1.0]: https://github.com/gosimd/unicode/releases/tag/v0.1.0
