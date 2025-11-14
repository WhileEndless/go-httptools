# Changelog

All notable changes to HTTPTools will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.0] - 2025-11-14

### Added
- **Automatic Chunked Transfer Encoding Decoding**: New opt-in feature for automatic decoding of chunked transfer encoding in HTTP responses
  - Added `ParseOptions` struct with `AutoDecodeChunked` and `PreserveChunkedTrailers` options
  - Added `response.ParseWithOptions()` function to parse responses with custom options
  - When `AutoDecodeChunked` is enabled:
    - Response body is automatically decoded from chunked format
    - Original chunked data is preserved in `RawBody` field
    - `Transfer-Encoding: chunked` header is removed
    - `Content-Length` header is added with decoded body size
    - `IsBodyChunked` is set to `false` after decoding
  - When `PreserveChunkedTrailers` is enabled (with `AutoDecodeChunked`):
    - HTTP trailers from chunked encoding are preserved as regular headers
  - Added comprehensive test suite for chunked encoding auto-decode feature
  - Added `examples/auto_decode_chunked.go` demonstrating the new feature

### Changed
- **Improved Response Body Parsing**: Refactored response parser to preserve raw body bytes exactly as received
  - Body parsing now uses `findHeaderEnd()` helper to locate header-body boundary
  - Raw body bytes are preserved without line-ending normalization
  - Fixes issues with chunked encoding preservation
- **Backwards Compatible**: Default behavior unchanged - `response.Parse()` continues to preserve chunked encoding
  - Users must explicitly opt-in to auto-decode feature via `ParseWithOptions()`

### Documentation
- Updated README.md with v1.1.0 badge and feature list
- Added API documentation for `ParseOptions` and `ParseWithOptions()`
- Added chunked encoding examples and usage patterns
- Updated version information throughout documentation

## [1.0.0] - 2025-11-13

### Added
- Initial release of HTTPTools library
- Fault-tolerant HTTP request/response parsing
- Header order preservation
- Non-standard header support
- Automatic decompression (gzip, deflate, brotli)
- Parse → Edit → Rebuild pipeline
- Exact format preservation for raw parsing
- Chunked transfer encoding support (manual decoding)
- Cookie parsing and manipulation
- Query parameter support
- HTTP/2 pseudo-headers support
- Comprehensive test suite
- Multiple usage examples

[1.1.0]: https://github.com/WhileEndless/go-httptools/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/WhileEndless/go-httptools/releases/tag/v1.0.0
