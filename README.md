# HTTPTools

[![Version](https://img.shields.io/badge/version-1.3.2-blue.svg)](https://github.com/WhileEndless/go-httptools)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/)

A robust HTTP request/response parser and editor for Go. Parse raw HTTP messages with fault tolerance, preserve exact formatting, and edit messages like Burp Suite.

## Features

- **Fault-tolerant parsing** of raw HTTP requests and responses
- **Header order preservation** for exact reconstruction
- **Non-standard header support** (`test:deneme`, malformed headers)
- **Automatic decompression** (gzip, deflate, brotli, zstd) with magic byte detection
- **Automatic chunked encoding decoding** (opt-in)
- **BuildOptions system** for flexible output control
- **HTTP/2 format support** with pseudo-headers
- **Search functionality** for requests/responses
- **Zstd (Zstandard) compression support** (NEW in v1.3.2)
- **Magic byte detection** for automatic compression identification (NEW in v1.3.2)
- **Parse → Edit → Rebuild** pipeline
- **Exact format preservation** (spacing, line endings, formatting)
- **Minimal external dependencies** (brotli and zstd compression libraries only)

## Quick Start

### Standard Parsing (with normalization)

```go
package main

import (
    "fmt"
    "github.com/WhileEndless/go-httptools/pkg/request"
)

func main() {
    rawReq := []byte(`GET /api/users HTTP/1.1
Host: example.com
test:deneme
Authorization: Bearer token123

`)

    req, err := request.Parse(rawReq)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Method: %s\n", req.Method)
    fmt.Printf("Custom header: %s\n", req.Headers.Get("test"))

    // Edit and rebuild
    req.Headers.Set("Authorization", "Bearer new-token")
    rebuilt := req.Build()
    fmt.Println(string(rebuilt))
}
```

### Raw Format Preservation (exact formatting)

```go
package main

import (
    "fmt"
    "github.com/WhileEndless/go-httptools/pkg/request"
)

func main() {
    // Malformed request with weird spacing
    original := []byte(`POST    /api/test   HTTP/1.1
Host:   example.com
test:deneme
Content-Type:  application/json

{  "data":  "test"  }`)

    // Parse preserving exact format
    rawReq, err := request.ParseRaw(original)
    if err != nil {
        panic(err)
    }

    // Access parsed data normally
    fmt.Printf("Method: %s\n", rawReq.Method)
    fmt.Printf("Custom header: %s\n", rawReq.Headers.Get("test"))

    // Edit one header
    rawReq.Headers.Set("Authorization", "Bearer token")

    // Rebuild with exact formatting preserved
    rebuilt := rawReq.BuildRaw()

    // Original spacing, line endings, formatting preserved!
    fmt.Printf("Identical: %t\n", string(original) == string(rebuilt))
}
```

## BuildOptions System (NEW in v1.3.0)

Full control over how requests/responses are built:

```go
import "github.com/WhileEndless/go-httptools/pkg/request"
import "github.com/WhileEndless/go-httptools/pkg/response"

// Check body state
if resp.IsCompressed() {
    fmt.Println("Compression:", resp.GetCompressionType())
}
if resp.IsChunked() {
    fmt.Println("Body is chunked")
}

// Build with full options
opts := request.BuildOptions{
    Compression:            request.CompressionGzip,  // None, Keep, Gzip, Deflate, Brotli, Zstd
    Chunked:                request.ChunkedRemove,    // Keep, Remove, Apply
    HTTPVersion:            request.HTTPVersion2,     // Keep, HTTP/1.1, HTTP/2
    UpdateContentLength:    true,
    UpdateContentEncoding:  true,
    UpdateTransferEncoding: true,
}
data, err := req.BuildWithOptions(opts)

// Convenience methods
data, _ := req.BuildNormalized()           // Decompressed, dechunked, HTTP/1.1
data, _ := req.BuildAsHTTP2()              // HTTP/2 format with pseudo-headers
data, _ := req.BuildDecompressed()         // Body decompressed
data, _ := req.BuildDechunked()            // Chunked encoding removed
data, _ := req.BuildWithCompression(request.CompressionGzip)  // Specific compression
```

### HTTP/2 Output Format

```go
req, _ := request.Parse(rawHTTP1Request)

// Build as HTTP/2 (auto-generates pseudo-headers)
http2Data, _ := req.BuildAsHTTP2()

// Output:
// :method: GET
// :scheme: https
// :authority: example.com
// :path: /api/users
// Content-Type: application/json
//
// {"data": "value"}
```

## Key Capabilities

### 1. **Fault Tolerance**
Handles malformed HTTP messages gracefully:
```go
malformed := []byte(`GET /path
Host: example.com
: empty-header-name
Invalid-Header-No-Colon
test:deneme

`)

req, _ := request.ParseRaw(malformed) // No error!
fmt.Println(req.Headers.Get("test")) // "deneme"
```

### 2. **Header Order Preservation & Positioning**
Maintains exact header ordering with precise control:
```go
rawReq := []byte(`POST / HTTP/1.1
Host: example.com
User-Agent: Mozilla/5.0
test:deneme
Content-Type: application/json

`)

req, _ := request.ParseRaw(rawReq)

// Add header at specific positions
req.Headers.SetAfter("Authorization", "Bearer token", "Host")     // After Host
req.Headers.SetBefore("Cookie", "session=123", "User-Agent")     // Before User-Agent
req.Headers.SetAt("X-Custom", "value", 0)                        // At index 0 (first)

// Result order: X-Custom, Host, Authorization, Cookie, User-Agent, test, Content-Type
// req.Headers.Get("test") returns "deneme" - custom headers preserved!
```

### 3. **Non-Standard Header Support**
Works with any header format:
```go
// All these work perfectly:
test:deneme                    // No space after colon
Weird Header Name: value       // Spaces in name
X-123-Numbers: value          // Numbers in name
header-without-value:         // Empty value
```

### 4. **Exact Format Preservation**
Preserves spacing, line endings, formatting:
```go
// Weird spacing preserved exactly
original := `GET     /path     HTTP/1.1
Host:example.com
test:deneme

`
rawReq, _ := request.ParseRaw([]byte(original))
rebuilt := rawReq.BuildRaw()
// string(original) == string(rebuilt)
```

### 5. **Flexible Line Endings**
Supports both `\r\n` and `\n` automatically:
```go
// Works with Unix line endings (\n)
unix := "GET / HTTP/1.1\nHost: example.com\ntest:deneme\n\n"

// Works with Windows line endings (\r\n)
windows := "GET / HTTP/1.1\r\nHost: example.com\r\ntest:deneme\r\n\r\n"

// Both parse correctly and preserve original format
```

## Burp Suite-like Editing

```go
editor := utils.NewRequestEditor(req)
modified := editor.
    SetMethod("PUT").
    SetURL("/api/users/123").
    AddHeader("Authorization", "Bearer token").
    UpdateHeader("Content-Type", "application/json").
    SetBodyString(`{"updated": true}`).
    AddQueryParam("force", "true").
    GetRequest()

rebuilt := modified.Build()
```

## Search Functionality

```go
import "github.com/WhileEndless/go-httptools/pkg/search"

// Search in request
results, _ := req.Search("token", search.DefaultOptions())
for _, r := range results.Results {
    fmt.Printf("Found in %s at position %d\n", r.Location, r.Position)
}

// Search only in headers or body
headerResults, _ := req.SearchHeaders("Authorization", true)  // case-insensitive
bodyResults, _ := req.SearchBody("password", false)

// Quick contains check
if req.Contains("secret", true) {
    fmt.Println("Found sensitive data!")
}

// Regex search
results, _ := req.SearchRegex(`Bearer\s+[\w-]+`)

// Search in response (same API)
results, _ := resp.Search("error", search.DefaultOptions())
if resp.Contains("404", false) {
    fmt.Println("Not found response")
}

// Replace in body
count, _ := req.ReplaceInBody("old_token", "new_token", search.DefaultOptions())
fmt.Printf("Replaced %d occurrences\n", count)
```

## API Overview

### Request Package
- `request.Parse([]byte)` - Standard parsing with normalization
- `request.ParseRaw([]byte)` - Exact format preservation parsing
- `req.Build()` - Standard rebuild
- `rawReq.BuildRaw()` - Exact format rebuild
- `req.BuildWithOptions(opts)` - Build with full control (NEW)
- `req.BuildAsHTTP2()` - Build as HTTP/2 format (NEW)
- `req.BuildNormalized()` - Build normalized HTTP/1.1 (NEW)
- `req.BuildDecompressed()` - Build with decompressed body (NEW)
- `req.BuildDechunked()` - Build without chunked encoding (NEW)
- `req.IsCompressed()` - Check if body is compressed
- `req.IsChunked()` - Check if body is chunked
- `req.Search(pattern, opts)` - Search in request
- `req.SearchHeaders(pattern, caseInsensitive)` - Search only in headers
- `req.SearchBody(pattern, caseInsensitive)` - Search only in body
- `req.SearchRegex(pattern)` - Search using regex
- `req.Contains(pattern, caseInsensitive)` - Quick contains check
- `req.ContainsRegex(pattern)` - Regex contains check
- `req.ReplaceInBody(pattern, replacement, opts)` - Replace in body

### Response Package
- `response.Parse([]byte)` - Parse with automatic decompression
- `response.ParseWithOptions([]byte, ParseOptions)` - Parse with custom options
- `resp.Build()` - Rebuild (compressed if original was compressed)
- `resp.BuildWithOptions(opts)` - Build with full control (NEW)
- `resp.BuildAsHTTP2()` - Build as HTTP/2 format (NEW)
- `resp.BuildNormalized()` - Build normalized HTTP/1.1 (NEW)
- `resp.BuildDecompressed()` - Rebuild with decompressed body
- `resp.BuildDechunked()` - Build without chunked encoding (NEW)
- `resp.IsCompressed()` - Check if body is compressed
- `resp.IsChunked()` - Check if body is chunked
- `resp.Search(pattern, opts)` - Search in response
- `resp.SearchHeaders(pattern, caseInsensitive)` - Search only in headers
- `resp.SearchBody(pattern, caseInsensitive)` - Search only in body
- `resp.SearchRegex(pattern)` - Search using regex
- `resp.Contains(pattern, caseInsensitive)` - Quick contains check
- `resp.ContainsRegex(pattern)` - Regex contains check
- `resp.ReplaceInBody(pattern, replacement, opts)` - Replace in body

#### Chunked Transfer Encoding

```go
// Default behavior: chunked body is preserved
resp, _ := response.Parse(chunkedResponse)
fmt.Println(resp.IsBodyChunked) // true

// Auto-decode chunked encoding
opts := response.ParseOptions{
    AutoDecodeChunked: true,
}
resp, _ := response.ParseWithOptions(chunkedResponse, opts)
fmt.Println(resp.IsBodyChunked)        // false (decoded)
fmt.Println(string(resp.Body))         // decoded, clean content
fmt.Println(string(resp.RawBody))      // original chunked data

// Preserve trailers from chunked encoding as headers
opts := response.ParseOptions{
    AutoDecodeChunked:       true,
    PreserveChunkedTrailers: true,
}
resp, _ := response.ParseWithOptions(chunkedResponse, opts)
// Trailers are now accessible via resp.Headers
```

### BuildOptions Reference

```go
type BuildOptions struct {
    // Compression: CompressionKeep, CompressionNone, CompressionGzip,
    //              CompressionDeflate, CompressionBrotli, CompressionZstd
    Compression CompressionMethod

    // Chunked: ChunkedKeep, ChunkedRemove, ChunkedApply
    Chunked ChunkedOption

    // ChunkSize for chunked encoding (0 = default 8192)
    ChunkSize int

    // HTTPVersion: HTTPVersionKeep, HTTPVersion11, HTTPVersion2
    HTTPVersion HTTPVersion

    // Auto-update headers based on body changes
    UpdateContentLength     bool  // default: true
    UpdateContentEncoding   bool  // default: true
    UpdateTransferEncoding  bool  // default: true

    // Line separator override (empty = use original)
    LineSeparator string

    // Preserve original header formatting
    PreserveOriginalHeaders bool  // default: true
}

// Preset options
opts := request.DefaultBuildOptions()     // Keep everything as-is
opts := request.DecompressedOptions()     // Decompress + dechunk
opts := request.NormalizedOptions()       // Full normalization
opts := request.HTTP2Options()            // HTTP/2 format
```

### Headers Package
- `headers.OrderedHeaders` - Standard headers with order preservation
- `headers.OrderedHeadersRaw` - Raw headers with exact format preservation
- **Positioning methods**: `SetAfter()`, `SetBefore()`, `SetAt()`
- Case-insensitive lookups, original case preservation

### Utils Package
- `RequestEditor` / `ResponseEditor` - Burp Suite-like editing
- `ValidateRequest()` / `ValidateResponse()` - Validation with warnings
- Standard library conversion utilities

### Search Package
- `search.SearchOptions` - Options for searching (Pattern, UseRegex, CaseInsensitive, etc.)
- `search.DefaultOptions()` - Default search options
- `search.SearchInHeaders` / `search.SearchInBody` - Location flags
- Search methods are available directly on Request and Response objects

### Compression Package (NEW in v1.3.2)
- `compression.DetectCompression(encoding)` - Detect from Content-Encoding header
- `compression.DetectByMagicBytes(data)` - Detect from magic bytes
- `compression.Compress(data, compressionType)` - Compress data
- `compression.CompressWithLevel(data, compressionType, level)` - Compress with level
- `compression.Decompress(data, compressionType)` - Decompress data
- `compression.DecompressAuto(data)` - Auto-detect and decompress
- `compression.IsSupported(encoding)` - Check if encoding is supported
- `compression.GetSupportedEncodings()` - Get list of supported encodings
- Supported types: `gzip`, `deflate`, `br` (brotli), `zstd`, `identity`
- Also supports aliases: `x-gzip`, `x-deflate`, `zstandard`

## Examples

See `examples/` directory for:
- `basic_parsing.go` - Basic parsing examples
- `editing_requests.go` - Request editing workflows
- `editing_responses.go` - Response editing with compression
- `fault_tolerance.go` - Malformed request handling
- `exact_preservation.go` - Format preservation demo
- `header_positioning.go` - Header positioning examples
- `burp_like_usage.go` - Burp Suite-like header management
- `auto_decode_chunked.go` - Automatic chunked encoding decoding

## Testing

```bash
# Run all tests
go test ./tests/...

# Run specific test suites
go test ./tests/unit/           # Unit tests
go test ./tests/integration/    # Integration tests

# Test examples
go run examples/exact_preservation.go
```

## Installation

```bash
go get github.com/WhileEndless/go-httptools
```

## Version Information

```go
package main

import (
    "fmt"
    "github.com/WhileEndless/go-httptools/pkg/version"
)

func main() {
    fmt.Println("HTTPTools version:", version.GetVersion()) // Output: 1.3.2
}
```

Current version: **1.3.2**

### Changelog

**v1.3.2**
- Add Zstd (Zstandard) compression support
- Add magic byte detection for automatic compression identification
- Add `DetectedCompression` field to Request and Response
- Add `CompressWithLevel()` for compression level control
- Add `DecompressAuto()` for auto-detecting and decompressing
- Add `IsSupported()` and `GetSupportedEncodings()` helper functions
- Support compression aliases: `x-gzip`, `x-deflate`, `zstandard`
- Parser now uses magic byte detection as fallback when headers are missing

**v1.3.1**
- Remove redundant `BuildHTTP2()` in favor of `BuildAsHTTP2()`
- Add missing `BuildDecompressed()` to response package

**v1.3.0**
- Add BuildOptions system for flexible output control
- Add HTTP/2 format support with auto-generated pseudo-headers
- Add search functionality for requests/responses
- Add `IsCompressed()` and `IsChunked()` methods
- Add convenience build methods: `BuildNormalized()`, `BuildAsHTTP2()`, etc.

**v1.2.x**
- Format preservation improvements
- Header formatting fixes

**v1.1.0**
- Automatic chunked encoding decoding (opt-in)
- ParseWithOptions for custom parsing behavior

## Use Cases

- **Security Testing Tools** - Parse and modify HTTP requests/responses
- **HTTP Proxies** - Intercept and edit traffic while preserving format
- **Web Scrapers** - Handle malformed HTTP responses gracefully
- **API Testing** - Edit requests with exact control over formatting
- **Protocol Research** - Analyze and reconstruct HTTP messages precisely
- **Burp Suite Extensions** - Go-based tools with similar capabilities

## Dependencies

- **Go 1.21+**
- **github.com/andybalholm/brotli** (for Brotli compression support)
- **github.com/klauspost/compress** (for Zstd compression support)

Minimal external dependencies - uses Go standard library for everything else.

