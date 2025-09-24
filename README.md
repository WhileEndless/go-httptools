# HTTPTools

[![Version](https://img.shields.io/badge/version-1.0.0-blue.svg)](https://github.com/WhileEndless/go-httptools)
[![Go](https://img.shields.io/badge/Go-1.21+-00ADD8.svg)](https://golang.org/)

A robust HTTP request/response parser and editor for Go. Parse raw HTTP messages with fault tolerance, preserve exact formatting, and edit messages like Burp Suite.

## Features

- **🔧 Fault-tolerant parsing** of raw HTTP requests and responses
- **📋 Header order preservation** for exact reconstruction  
- **🎯 Non-standard header support** (`test:deneme`, malformed headers)
- **🗜️ Automatic decompression** (gzip, deflate, brotli)
- **✏️ Parse → Edit → Rebuild** pipeline
- **📐 Exact format preservation** (spacing, line endings, formatting)
- **⚡ Zero external dependencies** (except brotli for compression)

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
// string(original) == string(rebuilt) ✅
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

## API Overview

### Request Package
- `request.Parse([]byte)` - Standard parsing with normalization
- `request.ParseRaw([]byte)` - Exact format preservation parsing
- `req.Build()` - Standard rebuild
- `rawReq.BuildRaw()` - Exact format rebuild

### Response Package  
- `response.Parse([]byte)` - Parse with automatic decompression
- `resp.Build()` - Rebuild (compressed if original was compressed)
- `resp.BuildDecompressed()` - Rebuild with decompressed body

### Headers Package
- `headers.OrderedHeaders` - Standard headers with order preservation
- `headers.OrderedHeadersRaw` - Raw headers with exact format preservation
- **Positioning methods**: `SetAfter()`, `SetBefore()`, `SetAt()`
- Case-insensitive lookups, original case preservation

### Utils Package
- `RequestEditor` / `ResponseEditor` - Burp Suite-like editing
- `ValidateRequest()` / `ValidateResponse()` - Validation with warnings
- Standard library conversion utilities

## Examples

See `examples/` directory for:
- `basic_parsing.go` - Basic parsing examples
- `editing_requests.go` - Request editing workflows
- `editing_responses.go` - Response editing with compression
- `fault_tolerance.go` - Malformed request handling
- `exact_preservation.go` - Format preservation demo
- `header_positioning.go` - Header positioning examples
- `burp_like_usage.go` - Burp Suite-like header management

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
    fmt.Println("HTTPTools version:", version.GetVersion()) // Output: 1.0.0
}
```

Current version: **1.0.0**

## Use Cases

- **Security Testing Tools** - Parse and modify HTTP requests/responses
- **HTTP Proxies** - Intercept and edit traffic while preserving format  
- **Web Scrapers** - Handle malformed HTTP responses gracefully
- **API Testing** - Edit requests with exact control over formatting
- **Protocol Research** - Analyze and reconstruct HTTP messages precisely
- **Burp Suite Extensions** - Go-based tools with similar capabilities

## Dependencies

- **Go 1.21+** 
- **github.com/andybalholm/brotli** (for Brotli compression support only)

Zero other external dependencies - uses only Go standard library.

